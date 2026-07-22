package http

import (
	"encoding/json"
	"html/template"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"linknest/internal/auth"
	"linknest/internal/config"
	"linknest/internal/metrics"
	"linknest/internal/models"
	"linknest/internal/store"
)

type Server struct {
	cfg     config.Config
	store   *store.Store
	metrics *metrics.Registry
	tmpl    *template.Template
}

func New(cfg config.Config, st *store.Store, registry *metrics.Registry, tmpl *template.Template) *Server {
	return &Server{cfg: cfg, store: st, metrics: registry, tmpl: tmpl}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("GET /", s.home)
	mux.HandleFunc("GET /up", s.up)
	mux.Handle("GET /metrics", s.metrics.Handler())
	mux.HandleFunc("GET /register", s.registerForm)
	mux.HandleFunc("POST /register", s.register)
	mux.HandleFunc("GET /login", s.loginForm)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("POST /logout", s.logout)
	mux.HandleFunc("GET /dashboard", s.requireAuth(s.dashboard))
	mux.HandleFunc("POST /profile", s.requireAuth(s.updateProfile))
	mux.HandleFunc("POST /links", s.requireAuth(s.createLink))
	mux.HandleFunc("POST /links/{id}/track_click", s.trackClick)
	mux.HandleFunc("GET /api/v1/events", s.requireAuth(s.apiEvents))
	mux.HandleFunc("GET /api/v1/status", s.requireAuth(s.apiStatus))
	mux.HandleFunc("GET /{slug}", s.publicProfile)
	return s.instrument(mux)
}

func (s *Server) instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.metrics.Inc("linknest_http_requests_total", 1)
		s.metrics.Observe("linknest_http_request_duration", time.Since(start))
	})
}

func (s *Server) up(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	s.render(w, "home.html", map[string]any{"User": user})
}

func (s *Server) registerForm(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "register.html", nil)
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	user, err := s.store.CreateUser(r.Context(), r.FormValue("email"), r.FormValue("password"), r.FormValue("username"), r.FormValue("first_name"), r.FormValue("last_name"))
	if err != nil {
		s.render(w, "register.html", map[string]any{"Error": err.Error()})
		return
	}
	sessionID, err := s.store.CreateSession(r.Context(), user.ID, clientIP(r), r.UserAgent())
	if err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}
	auth.SetSessionCookie(w, sessionID, s.cfg.SessionSecret)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) loginForm(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "login.html", nil)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	user, err := s.store.Authenticate(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		s.render(w, "login.html", map[string]any{"Error": "Try another email or password."})
		return
	}
	sessionID, err := s.store.CreateSession(r.Context(), user.ID, clientIP(r), r.UserAgent())
	if err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}
	auth.SetSessionCookie(w, sessionID, s.cfg.SessionSecret)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		if sessionID, ok := auth.Verify(cookie.Value, s.cfg.SessionSecret); ok {
			_ = s.store.DeleteSession(r.Context(), sessionID)
		}
	}
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request, user models.User) {
	dashboard, err := s.store.Dashboard(r.Context(), user)
	if err != nil {
		http.Error(w, "dashboard error", http.StatusInternalServerError)
		return
	}
	s.render(w, "dashboard.html", dashboard)
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, user models.User) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if err := s.store.UpdateProfile(r.Context(), user.ID, r.FormValue("first_name"), r.FormValue("last_name"), r.FormValue("bio")); err != nil {
		http.Error(w, "profile update failed", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) createLink(w http.ResponseWriter, r *http.Request, user models.User) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	public := r.FormValue("public") == "on"
	if err := s.store.CreateLink(r.Context(), user.ID, r.FormValue("title"), safeURL(r.FormValue("url")), public); err != nil {
		http.Error(w, "link create failed", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) trackClick(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	link, err := s.store.LinkForTracking(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	now := time.Now().UTC()
	event := models.ClickEvent{
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
		UserID:         link.UserID,
		LinkID:         link.ID,
		Referrer:       r.Referer(),
		UserAgent:      r.UserAgent(),
		IPAddress:      clientIP(r),
		CountryCode:    headerOr(r, "CF-IPCountry", "US"),
		DeviceType:     deviceType(r.UserAgent()),
		BrowserName:    browserName(r.UserAgent()),
		EventTime:      now,
	}
	if event.IdempotencyKey == "" {
		event.IdempotencyKey = auth.IdempotencyKey(event.LinkID, event.IPAddress, event.UserAgent, event.Referrer, now)
	}
	inserted, err := s.store.IngestClick(r.Context(), event)
	if err != nil {
		http.Error(w, "tracking failed", http.StatusInternalServerError)
		return
	}
	if inserted {
		s.metrics.Inc("linknest_click_events_ingested_total", 1)
	} else {
		s.metrics.Inc("linknest_click_events_deduplicated_total", 1)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) publicProfile(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if strings.Contains(slug, ".") || reservedSlug(slug) {
		http.NotFound(w, r)
		return
	}
	user, err := s.store.UserBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	links, err := s.store.LinksForUser(r.Context(), user.ID, true, 100, 0)
	if err != nil {
		http.Error(w, "profile error", http.StatusInternalServerError)
		return
	}
	s.render(w, "profile.html", map[string]any{"User": user, "Links": links})
}

func (s *Server) apiEvents(w http.ResponseWriter, r *http.Request, user models.User) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	events, err := s.store.APIHistory(r.Context(), user.ID, limit, after)
	if err != nil {
		http.Error(w, "history error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"events": events, "next_after": nextAfter(events)})
}

func (s *Server) apiStatus(w http.ResponseWriter, r *http.Request, _ models.User) {
	summary, err := s.store.ExplainIndexes(r.Context())
	if err != nil {
		http.Error(w, "status error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"status": "ok", "database": summary})
}

func (s *Server) currentUser(r *http.Request) (models.User, bool) {
	cookie, err := r.Cookie(auth.CookieName)
	if err != nil {
		return models.User{}, false
	}
	sessionID, ok := auth.Verify(cookie.Value, s.cfg.SessionSecret)
	if !ok {
		return models.User{}, false
	}
	user, err := s.store.UserBySession(r.Context(), sessionID)
	return user, err == nil
}

func (s *Server) requireAuth(next func(http.ResponseWriter, *http.Request, models.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.currentUser(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r, user)
	}
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func nextAfter(events []models.ClickEvent) int64 {
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].ID
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func headerOr(r *http.Request, key string, fallback string) string {
	value := r.Header.Get(key)
	if value == "" {
		return fallback
	}
	return value
}

func deviceType(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "mobile"), strings.Contains(lower, "android"), strings.Contains(lower, "iphone"):
		return "mobile"
	case strings.Contains(lower, "tablet"), strings.Contains(lower, "ipad"):
		return "tablet"
	default:
		return "desktop"
	}
}

func browserName(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "edg"):
		return "Edge"
	case strings.Contains(lower, "chrome"):
		return "Chrome"
	case strings.Contains(lower, "firefox"):
		return "Firefox"
	case strings.Contains(lower, "safari"):
		return "Safari"
	default:
		return "Other"
	}
}

func safeURL(value string) string {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "mailto:") || strings.HasPrefix(lower, "tel:") || strings.HasPrefix(value, "/") {
		return value
	}
	return "https://" + value
}

func reservedSlug(slug string) bool {
	switch slug {
	case "login", "logout", "register", "dashboard", "links", "api", "metrics", "static", "up":
		return true
	default:
		return false
	}
}
