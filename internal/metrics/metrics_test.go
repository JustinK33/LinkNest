package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestRenderPrometheusMetrics(t *testing.T) {
	registry := New()
	registry.Inc("linknest_click_events_ingested_total", 2)
	registry.Gauge("linknest_worker_hourly_rows_processed", 5)
	registry.Observe("linknest_worker_hourly_duration", 10*time.Millisecond)
	rendered := registry.Render()
	for _, want := range []string{
		"linknest_click_events_ingested_total 2",
		"linknest_worker_hourly_rows_processed 5",
		"linknest_worker_hourly_duration_seconds",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("metrics output missing %q:\n%s", want, rendered)
		}
	}
}

func BenchmarkMetricsIncrement(b *testing.B) {
	registry := New()
	for i := 0; i < b.N; i++ {
		registry.Inc("linknest_click_events_ingested_total", 1)
	}
}
