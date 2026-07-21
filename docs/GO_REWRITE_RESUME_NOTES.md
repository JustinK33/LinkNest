# Go Rewrite Resume Notes

## Summary

LinkNest is implemented as a Go and PostgreSQL service.
The application supports authenticated users, public link profiles, dashboard management, click tracking, and analytics.
It also adds systems and data-engineering concepts that are natural for a write-heavy analytics product.

## Implemented Improvements

### 1. Storage Engine Concepts

Implemented an append-only `click_events` table that acts as a durable event log for replayable analytics.
Each click event is written once and later aggregated into hourly and daily reporting tables.
This gives the project a clear event-log and recovery story without inventing a fake database engine.

Added `analytics_snapshots` to persist daily aggregate state with a `source_event_max_id` high-water mark.
This supports faster recovery and auditability after worker crashes because daily state can be compared against the raw event stream.

Resume bullet:

- Built append-only event logs and daily analytics snapshots in PostgreSQL, enabling replayable click-history recovery and auditable aggregate state.

### 2. Database Engineering

The PostgreSQL schema uses composite indexes around the actual query paths:

- `idx_links_user_position` for dashboard ordering.
- `idx_links_user_public_position` for public profile rendering.
- `idx_links_user_click_count` for top-link ranking.
- `idx_click_events_user_time` for user analytics windows.
- `idx_click_events_link_time` for per-link analytics windows.
- `idx_click_events_link_time_ip` for unique visitor calculations.
- `idx_hourly_stats_user_hour` for dashboard rollups.

Writes use transactions where consistency matters.
Click ingestion inserts the raw event and increments the link counter in one transaction.
Link creation calculates the next position and inserts the row in one transaction.

Batching is implemented in the aggregators using `INSERT ... SELECT ... GROUP BY ... ON CONFLICT DO UPDATE`.
This lets PostgreSQL aggregate many events in one statement instead of pushing row-by-row work through application loops.

Connection pooling is configurable with `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, and `DB_CONN_MAX_LIFETIME_SECONDS`.

Resume bullets:

- Designed PostgreSQL composite indexes for profile lookup, event ingestion, unique-visitor analytics, and top-link ranking.
- Batched hourly and daily analytics rollups with transactional SQL upserts, reducing per-event reporting work while preserving consistency.
- Tuned PostgreSQL connection pooling with configurable open, idle, and lifetime limits for concurrent request and worker execution.

### 3. Idempotency

`POST /links/{id}/track_click` accepts an `Idempotency-Key` header.
When the client does not provide one, the service derives a deterministic key from link ID, IP address, user agent, referrer, and a one-minute time bucket.
The database enforces uniqueness with `click_events.idempotency_key`.
Duplicate submissions are ignored with `ON CONFLICT DO NOTHING`.

Resume bullet:

- Prevented duplicate click ingestion with idempotency keys, deterministic fallback tokens, and atomic PostgreSQL `ON CONFLICT DO NOTHING` writes.

### 4. API Design

The Go service exposes product pages and API endpoints:

- `POST /links/{id}/track_click` for asynchronous click ingestion.
- `GET /api/v1/events?limit=50&after=0` for cursor-paginated event history.
- `GET /api/v1/status` for service and database status.
- `GET /metrics` for Prometheus-compatible metrics.
- `GET /up` for health checks.

Resume bullet:

- Designed Go REST APIs for asynchronous click ingestion, cursor-paginated event history, service status, health checks, and metrics export.

### 5. Observability

The service exposes Prometheus-compatible metrics from `/metrics`.
Implemented counters, gauges, and duration observations for:

- HTTP request count.
- HTTP request duration.
- Click events ingested.
- Click events deduplicated.
- Worker failures.
- Hourly and daily worker run counts.
- Worker rows processed.
- Worker durations.
- Retention cleanup rows deleted.

Resume bullet:

- Instrumented Prometheus-compatible metrics exposing request volume, ingestion deduplication, worker throughput, processing latency, and retention cleanup.

### 6. Load Testing

Added `loadtest/k6-clicks.js`.
The script simulates concurrent users sending click events with unique idempotency keys and configurable virtual users.

Example command:

```sh
k6 run -e BASE_URL=http://localhost:8081 -e LINK_ID=1 -e VUS=100 -e DURATION=30s loadtest/k6-clicks.js
```

Measured k6 result against the Docker Compose Go/PostgreSQL stack:

- `100` concurrent users.
- `28,835` successful tracking requests in `30s`.
- `957.89 events/sec`.
- `21.92 ms p95`.
- `0%` request failures.
- `28,835` persisted click events.
- `28,835` unique idempotency keys.
- Hourly SQL rollup processed `28,835` events in `36.415 ms`.
- Daily SQL rollup processed `28,835` events in `40.937 ms`.

Resume bullet:

- Sustained `957.89 events/sec` at `21.92 ms p95` under `100` concurrent users using composite indexes, batched SQL rollups, and tuned PostgreSQL connection pooling.

### 7. Security

Sensitive runtime configuration is environment-driven.
`.env` files remain ignored.
Sessions are signed with HMAC-SHA256.
Passwords are stored with bcrypt.
Public tracking only accepts public links.
URL handling normalizes normal domains to `https://` and allows only `http`, `https`, `mailto`, `tel`, and relative paths.

Resume bullet:

- Hardened authentication and request handling with bcrypt password storage, HMAC-signed session cookies, environment-based secrets, and public/private link access controls.

### 8. Infrastructure

The project now uses a Docker multi-stage build that compiles a static Go binary and runs it in a small Alpine runtime image.
Docker Compose starts the Go service and PostgreSQL with health checks.

Resume bullet:

- Deployed a Go/PostgreSQL stack with Docker multi-stage builds, service health checks, and environment-driven runtime configuration.

### 9. Backpressure

The current implementation does not yet include bounded internal queues because click ingestion writes directly to PostgreSQL.
The natural next improvement is to add a bounded in-process channel or external queue between HTTP ingestion and persistence.
Metrics are already in place to expose queue depth once this is added.

Future resume bullet:

- Added bounded ingestion queues and producer throttling to protect workers and PostgreSQL under burst traffic.

## Test And Benchmark Evidence

Command run locally:

```sh
env GOCACHE=/private/tmp/linknest-go-cache GOPATH=/private/tmp/linknest-go go test ./... -bench=. -benchmem
```

Result:

- Unit tests passed for auth, session signing, slug generation, idempotency key bucketing, URL normalization, device/browser parsing, metrics rendering, and store construction.
- `BenchmarkIdempotencyKey`: about `229 ns/op`, `280 B/op`, `9 allocs/op` on Apple M4.
- `BenchmarkMetricsIncrement`: about `13.76 ns/op`, `0 B/op`, `0 allocs/op` on Apple M4.
- k6 click-ingestion test: `957.89 events/sec`, `21.92 ms p95`, `0%` failures under `100` concurrent users.
- PostgreSQL verification: `28,835` click events, `28,835` unique idempotency keys, and `28,835` cached link clicks.
- SQL rollup verification: hourly aggregation completed in `36.415 ms`, and daily aggregation completed in `40.937 ms`.

## Strong Resume Bullets

- Built a Go and PostgreSQL link-in-bio analytics app with authenticated dashboards, public profiles, click tracking, and analytics rollups.
- Built append-only click-event logging with idempotency keys, transactional counter updates, and replayable analytics history.
- Designed PostgreSQL schemas with composite indexes, foreign keys, and SQL upserts for high-volume event ingestion and reporting queries.
- Implemented batched hourly and daily analytics workers using `INSERT ... SELECT` aggregation, daily snapshots, and 90-day raw event retention.
- Exposed Prometheus-compatible metrics for request volume, ingestion deduplication, worker throughput, processing latency, and cleanup activity.
- Added Go unit tests and benchmarks plus a k6 load-test harness for validating latency, throughput, and duplicate-event protection.
- Sustained `957.89 events/sec` at `21.92 ms p95` under `100` concurrent users using composite indexes, batched SQL rollups, and tuned PostgreSQL connection pooling.
