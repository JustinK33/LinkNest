package worker

import (
	"context"
	"log"
	"time"

	"linknest/metrics"
	"linknest/store"
)

type Manager struct {
	store   *store.Store
	metrics *metrics.Registry
}

func New(st *store.Store, registry *metrics.Registry) *Manager {
	return &Manager{store: st, metrics: registry}
}

func (m *Manager) Start(ctx context.Context) {
	hourly := time.NewTicker(5 * time.Minute)
	daily := time.NewTicker(30 * time.Minute)
	defer hourly.Stop()
	defer daily.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hourly.C:
			m.runHourly(ctx)
		case <-daily.C:
			m.runDaily(ctx)
		}
	}
}

func (m *Manager) runHourly(ctx context.Context) {
	start := time.Now()
	rows, err := m.store.AggregatePreviousHour(ctx)
	m.metrics.Observe("linknest_worker_hourly_duration", time.Since(start))
	if err != nil {
		log.Printf("hourly aggregation failed: %v", err)
		m.metrics.Inc("linknest_worker_failures_total", 1)
		return
	}
	m.metrics.Inc("linknest_worker_hourly_runs_total", 1)
	m.metrics.Gauge("linknest_worker_hourly_rows_processed", rows)
}

func (m *Manager) runDaily(ctx context.Context) {
	start := time.Now()
	rows, err := m.store.AggregateDay(ctx, time.Now().UTC().Add(-24*time.Hour))
	m.metrics.Observe("linknest_worker_daily_duration", time.Since(start))
	if err != nil {
		log.Printf("daily aggregation failed: %v", err)
		m.metrics.Inc("linknest_worker_failures_total", 1)
		return
	}
	if deleted, err := m.store.CleanupOldClicks(ctx, 90); err == nil {
		m.metrics.Gauge("linknest_worker_retention_rows_deleted", deleted)
	}
	m.metrics.Inc("linknest_worker_daily_runs_total", 1)
	m.metrics.Gauge("linknest_worker_daily_rows_processed", rows)
}
