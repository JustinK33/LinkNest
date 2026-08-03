// Package handler wraps LinkNest as a single Vercel serverless function.
// vercel.json rewrites every request here; internal/http/server.go does the
// actual routing.
package handler

import (
	"context"
	"log"
	"net/http"
	"sync"

	"linknest/app"
	"linknest/config"
	"linknest/db"
)

var (
	mu     sync.Mutex
	routes http.Handler
)

// ensureRoutes lazily connects and migrates on first use, retrying on every
// call until it succeeds. TiDB Cloud Serverless auto-pauses when idle, so an
// early attempt can time out waking it back up; unlike sync.Once, a failure
// here doesn't permanently wedge this warm container.
func ensureRoutes() (http.Handler, error) {
	mu.Lock()
	defer mu.Unlock()
	if routes != nil {
		return routes, nil
	}
	cfg := config.Load()
	pool, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(context.Background(), pool); err != nil {
		return nil, err
	}
	routes = app.New(cfg, pool).Routes()
	return routes, nil
}

// Handler is the Vercel Go runtime entrypoint (github.com/vercel/vercel/blob/main/DEVELOPING_A_RUNTIME.md).
// Background workers (hourly/daily rollups) don't run here - serverless
// functions aren't long-lived enough for a ticker. Run cmd/linknest for that.
func Handler(w http.ResponseWriter, r *http.Request) {
	h, err := ensureRoutes()
	if err != nil {
		log.Printf("startup failed: %v", err)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	h.ServeHTTP(w, r)
}
