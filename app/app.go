package app

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"linknest/config"
	apphttp "linknest/http"
	"linknest/metrics"
	"linknest/store"
	"linknest/web"
	"linknest/worker"
)

type App struct {
	cfg     config.Config
	store   *store.Store
	metrics *metrics.Registry
	workers *worker.Manager
	pages   map[string]*template.Template
}

// Funcs are the template helpers every page is parsed with. Exported so dev
// tooling renders pages exactly the way the server does.
var Funcs = template.FuncMap{
	"initials": func(first, last string) string {
		var b strings.Builder
		if first != "" {
			b.WriteRune([]rune(first)[0])
		}
		if last != "" {
			b.WriteRune([]rune(last)[0])
		}
		return strings.ToUpper(b.String())
	},
	// sub exists because text/template has no arithmetic, and the dashboard needs
	// to know which link is last so it can disable its "move down" arrow.
	"sub": func(a, b int) int { return a - b },
	// host renders the destination domain under a public link, so a visitor can
	// see where a link goes before tapping it.
	"host": func(raw string) string {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return ""
		}
		return strings.TrimPrefix(parsed.Host, "www.")
	},
}

// PageNames lists every standalone page template. Each is parsed together
// with layout.html into its own *template.Template so their "content"
// blocks don't collide with one another.
var PageNames = []string{"home.html", "login.html", "register.html", "dashboard.html", "profile.html", "error.html"}

func New(cfg config.Config, db *sql.DB) *App {
	registry := metrics.New()
	st := store.New(db)
	pages := make(map[string]*template.Template, len(PageNames))
	for _, name := range PageNames {
		pages[name] = template.Must(template.New(name).Funcs(Funcs).ParseFS(web.Templates, "templates/layout.html", "templates/"+name))
	}
	return &App{
		cfg:     cfg,
		store:   st,
		metrics: registry,
		workers: worker.New(st, registry),
		pages:   pages,
	}
}

func (a *App) Routes() http.Handler {
	server := apphttp.New(a.cfg, a.store, a.metrics, a.pages)
	return server.Routes()
}

func (a *App) Workers() *worker.Manager {
	return a.workers
}
