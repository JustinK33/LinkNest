package app

import (
	"database/sql"
	"html/template"
	"net/http"
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

var funcs = template.FuncMap{
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
}

// pageNames lists every standalone page template. Each is parsed together
// with layout.html into its own *template.Template so their "content"
// blocks don't collide with one another.
var pageNames = []string{"home.html", "login.html", "register.html", "dashboard.html", "profile.html"}

func New(cfg config.Config, db *sql.DB) *App {
	registry := metrics.New()
	st := store.New(db)
	pages := make(map[string]*template.Template, len(pageNames))
	for _, name := range pageNames {
		pages[name] = template.Must(template.New(name).Funcs(funcs).ParseFS(web.Templates, "templates/layout.html", "templates/"+name))
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
