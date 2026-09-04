package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"linknest/auth"
	"linknest/config"
	"linknest/metrics"
	"linknest/models"
	"linknest/store"
	"linknest/web"
)

type Server struct {
	cfg     config.Config
	store   *store.Store
	metrics *metrics.Registry
	pages   map[string]*template.Template
}

func New(cfg config.Config, st *store.Store, registry *metrics.Registry, pages map[string]*template.Template) *Server {
	return &Server{cfg: cfg, store: st, metrics: registry, pages: pages}
}

func (s *Server) Routes() http.Handler {
	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))
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
	// "GET /" is the mux catch-all, so anything with extra path segments lands
	// here too. Without this a URL like /a/b would render the landing page.
	if r.URL.Path != "/" {
		s.notFound(w, r)
		return
	}
	s.render(w, r, "home.html", nil)
}

func (s *Server) registerForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "register.html", map[string]any{})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "We couldn't read that form. Please try again.")
		return
	}
	user, err := s.store.CreateUser(r.Context(), r.FormValue("email"), r.FormValue("password"), r.FormValue("username"), r.FormValue("first_name"), r.FormValue("last_name"))
	if err != nil {
		// Hand back what they typed so a single conflict doesn't wipe the form.
		s.render(w, r, "register.html", map[string]any{
			"Error":     userMessage(err, "register", "We couldn't create your account just now. Please try again."),
			"Username":  r.FormValue("username"),
			"FirstName": r.FormValue("first_name"),
			"LastName":  r.FormValue("last_name"),
			"Email":     r.FormValue("email"),
		})
		return
	}
	sessionID, err := s.store.CreateSession(r.Context(), user.ID, clientIP(r), r.UserAgent())
	if err != nil {
		log.Printf("register: create session: %v", err)
		s.renderError(w, r, http.StatusInternalServerError, "Your account was created but we couldn't sign you in. Try logging in.")
		return
	}
	auth.SetSessionCookie(w, sessionID, s.cfg.SessionSecret)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) loginForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "login.html", map[string]any{})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "We couldn't read that form. Please try again.")
		return
	}
	user, err := s.store.Authenticate(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		s.render(w, r, "login.html", map[string]any{
			"Error": "Try another email or password.",
			"Email": r.FormValue("email"),
		})
		return
	}
	sessionID, err := s.store.CreateSession(r.Context(), user.ID, clientIP(r), r.UserAgent())
	if err != nil {
		log.Printf("login: create session: %v", err)
		s.renderError(w, r, http.StatusInternalServerError, "We couldn't sign you in just now. Please try again.")
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
	auth.SetFlash(w, "notice", "You're signed out.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request, user models.User) {
	dashboard, err := s.store.Dashboard(r.Context(), user)
	if err != nil {
		log.Printf("dashboard: %v", err)
		s.renderError(w, r, http.StatusInternalServerError, "We couldn't load your dashboard. Please try again.")
		return
	}
	s.render(w, r, "dashboard.html", dashboard)
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, user models.User) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "We couldn't read that form. Please try again.")
		return
	}
	err := s.store.UpdateProfile(r.Context(), user.ID, r.FormValue("first_name"), r.FormValue("last_name"), r.FormValue("bio"))
	s.flashResult(w, err, "update profile", "Profile saved.", "We couldn't save your profile. Please try again.")
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) createLink(w http.ResponseWriter, r *http.Request, user models.User) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "We couldn't read that form. Please try again.")
		return
	}
	title := r.FormValue("title")
	err := s.store.CreateLink(r.Context(), user.ID, title, safeURL(r.FormValue("url")), r.FormValue("public") == "on")
	s.flashResult(w, err, "create link", "Added \""+title+"\" to your profile.", "We couldn't add that link. Please try again.")
	http.Redirect(w, r, "/dashboard#links", http.StatusSeeOther)
}

// flashResult turns a store result into the one-shot message shown after the
// redirect. A models.UserError is already written for a person; anything else is
// logged and replaced, so driver text never reaches the page.
func (s *Server) flashResult(w http.ResponseWriter, err error, op string, success string, failure string) {
	if err == nil {
		auth.SetFlash(w, "notice", success)
		return
	}
	auth.SetFlash(w, "alert", userMessage(err, op, failure))
}

func userMessage(err error, op string, fallback string) string {
	var friendly models.UserError
	if errors.As(err, &friendly) {
		return friendly.Error()
	}
	log.Printf("%s: %v", op, err)
	return fallback
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
		s.notFound(w, r)
		return
	}
	user, err := s.store.UserBySlug(r.Context(), slug)
	if err != nil {
		s.notFound(w, r)
		return
	}
	links, err := s.store.LinksForUser(r.Context(), user.ID, true, 100, 0)
	if err != nil {
		log.Printf("public profile %s: %v", slug, err)
		s.renderError(w, r, http.StatusInternalServerError, "We couldn't load this profile. Please try again.")
		return
	}
	s.render(w, r, "profile.html", map[string]any{"User": user, "Links": links})
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	s.renderError(w, r, http.StatusNotFound, "We couldn't find that page. The link may have changed or the profile may no longer exist.")
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
			// Without this the redirect looks like the login page appeared for no
			// reason, which is indistinguishable from a bug to whoever hit it.
			auth.SetFlash(w, "alert", "Please sign in to continue.")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r, user)
	}
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	s.renderStatus(w, r, http.StatusOK, name, data)
}

// renderStatus buffers the template before writing anything. Rendering straight
// to the ResponseWriter would commit a 200 and partial HTML before a mid-render
// failure could be reported, so the error path below could never work.
func (s *Server) renderStatus(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	user, _ := s.currentUser(r)
	page := struct {
		CurrentUser models.User
		Flash       *auth.Flash
		SiteURL     string
		Data        any
	}{CurrentUser: user, Flash: auth.TakeFlash(w, r), SiteURL: siteURL(r), Data: data}

	var buf bytes.Buffer
	if err := s.pages[name].ExecuteTemplate(&buf, "layout", page); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "Something went wrong on our end.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

// renderError shows a styled error page inside the app layout instead of
// dropping the user onto a bare plain-text response.
func (s *Server) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	s.renderStatus(w, r, status, "error.html", map[string]any{
		"Status":  status,
		"Title":   http.StatusText(status),
		"Message": message,
	})
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

// siteURL is the request origin, with no trailing slash. Social crawlers won't
// resolve a relative og:image or og:url, so templates need to build absolute
// ones. Behind Vercel's proxy the scheme only shows up in X-Forwarded-Proto.
func siteURL(r *http.Request) string {
	scheme := "https"
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	} else if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host
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
