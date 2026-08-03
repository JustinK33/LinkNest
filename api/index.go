// Package handler wraps LinkNest as a single Vercel serverless function.
// vercel.json rewrites every request here; internal/http/server.go does the
// actual routing.
package handler

import (
	"context"
	"log"
	"net/http"
	"sync"

	"linknest/internal/app"
	"linknest/internal/config"
	"linknest/internal/db"
)

var (
	once     sync.Once
	routes   http.Handler
	setupErr error
)

func setup() {
	cfg := config.Load()
	pool, err := db.Open(cfg)
	if err != nil {
		setupErr = err
		return
	}
	if err := db.Migrate(context.Background(), pool); err != nil {
		setupErr = err
		return
	}
	routes = app.New(cfg, pool).Routes()
}

// Handler is the Vercel Go runtime entrypoint (github.com/vercel/vercel/blob/main/DEVELOPING_A_RUNTIME.md).
// Background workers (hourly/daily rollups) don't run here - serverless
// functions aren't long-lived enough for a ticker. Run cmd/linknest for that.
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(setup)
	if setupErr != nil {
		log.Printf("startup failed: %v", setupErr)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	routes.ServeHTTP(w, r)
}
