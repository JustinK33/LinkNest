package app

import (
	"database/sql"
	"html/template"
	"net/http"

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
	tmpl    *template.Template
}

func New(cfg config.Config, db *sql.DB) *App {
	registry := metrics.New()
	st := store.New(db)
	tmpl := template.Must(template.ParseFS(web.Templates, "templates/*.html"))
	return &App{
		cfg:     cfg,
		store:   st,
		metrics: registry,
		workers: worker.New(st, registry),
		tmpl:    tmpl,
	}
}

func (a *App) Routes() http.Handler {
	server := apphttp.New(a.cfg, a.store, a.metrics, a.tmpl)
	return server.Routes()
}

func (a *App) Workers() *worker.Manager {
	return a.workers
}
