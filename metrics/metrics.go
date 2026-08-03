package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type Registry struct {
	mu       sync.RWMutex
	counters map[string]int64
	gauges   map[string]int64
	timings  map[string][]time.Duration
}

func New() *Registry {
	return &Registry{
		counters: map[string]int64{},
		gauges:   map[string]int64{},
		timings:  map[string][]time.Duration{},
	}
}

func (r *Registry) Inc(name string, delta int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] += delta
}

func (r *Registry) Gauge(name string, value int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = value
}

func (r *Registry) Observe(name string, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.timings[name] = append(r.timings[name], duration)
	if len(r.timings[name]) > 1024 {
		r.timings[name] = r.timings[name][len(r.timings[name])-1024:]
	}
}

func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprint(w, r.Render())
	})
}

func (r *Registry) Render() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for name := range r.counters {
		names = append(names, name)
	}
	for name := range r.gauges {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		if value, ok := r.counters[name]; ok {
			fmt.Fprintf(&b, "# TYPE %s counter\n%s %d\n", name, name, value)
		}
		if value, ok := r.gauges[name]; ok {
			fmt.Fprintf(&b, "# TYPE %s gauge\n%s %d\n", name, name, value)
		}
	}
	for name, values := range r.timings {
		if len(values) == 0 {
			continue
		}
		var total time.Duration
		for _, value := range values {
			total += value
		}
		avg := total.Seconds() / float64(len(values))
		fmt.Fprintf(&b, "# TYPE %s_seconds gauge\n%s_seconds %.6f\n", name, name, avg)
	}
	return b.String()
}
