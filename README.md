# LinkNest

[![CI](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml/badge.svg)](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml)

LinkNest is a Go and PostgreSQL link-in-bio platform with public profiles, authenticated dashboard management, click tracking, analytics rollups, metrics, and load-test hooks.
The application was rewritten from Rails into a smaller Go service with explicit database and systems-engineering primitives.

## Tech Stack

- Go 1.26
- PostgreSQL
- `net/http`
- `database/sql`
- `pgx`
- Docker and Docker Compose
- Prometheus-compatible metrics
- k6 load testing

## Architecture

The Go service serves both HTML pages and REST-style API endpoints.
PostgreSQL stores users, signed sessions, links, append-only click events, hourly rollups, daily rollups, analytics snapshots, and worker run history.
Background workers run inside the Go process and batch aggregation work with SQL upserts.

Core paths:

- `GET /` - landing page
- `GET /register` and `POST /register` - account creation
- `GET /login` and `POST /login` - session login
- `GET /dashboard` - authenticated profile, link, and analytics dashboard
- `GET /{slug}` - public profile
- `POST /links/{id}/track_click` - idempotent click ingestion
- `GET /api/v1/events?limit=50&after=0` - cursor-paginated event history
- `GET /api/v1/status` - database status summary
- `GET /metrics` - Prometheus-compatible metrics
- `GET /up` - health check

## Systems Features

- Append-only `click_events` table for replayable analytics history.
- Idempotent click ingestion with `Idempotency-Key` support and deterministic fallback keys.
- Transactional writes for link creation and click ingestion.
- Composite PostgreSQL indexes for profile lookup, public link listing, event ingestion, and analytics ranges.
- Batched hourly and daily aggregations with `INSERT ... SELECT ... ON CONFLICT DO UPDATE`.
- Analytics snapshots that capture daily totals and source event high-water marks.
- Connection pool tuning through `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, and `DB_CONN_MAX_LIFETIME_SECONDS`.
- Prometheus-compatible counters, gauges, and duration metrics.
- k6 script for concurrent click-ingestion load testing.

## Local Development

Run the full stack:

```sh
docker compose up --build
```

Open the app:

```text
http://localhost:8080
```

The Compose stack starts PostgreSQL and the Go web service.
The Go process applies `migrations/001_init_postgres.sql` on boot.

## Environment

Use environment variables for all sensitive values.
Never commit `.env` files.

Important variables:

```sh
ADDR=:8080
DATABASE_URL=postgres://linknest:linknest@postgres:5432/linknest?sslmode=disable
SESSION_SECRET=replace-with-a-long-random-secret
DB_MAX_OPEN_CONNS=20
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME_SECONDS=300
```

## Testing

Run unit tests and benchmarks:

```sh
env GOCACHE=/private/tmp/linknest-go-cache GOPATH=/private/tmp/linknest-go go test ./... -bench=. -benchmem
```

Current local benchmark sample on Apple M4:

- `BenchmarkIdempotencyKey`: about `229 ns/op`
- `BenchmarkMetricsIncrement`: about `13.76 ns/op`, `0 B/op`, `0 allocs/op`

Run the k6 click-ingestion load test after creating a user and public link:

```sh
k6 run -e BASE_URL=http://localhost:8080 -e LINK_ID=1 -e VUS=50 -e DURATION=30s loadtest/k6-clicks.js
```

## Project Layout

```text
cmd/linknest/          Go entrypoint
internal/auth/         Passwords, signed sessions, slugs, idempotency keys
internal/config/       Environment-based configuration
internal/db/           PostgreSQL connection and migration runner
internal/http/         HTTP routes and handlers
internal/metrics/      Prometheus-compatible in-memory metrics registry
internal/models/       Domain structs
internal/store/        SQL queries, transactions, upserts, aggregation
internal/worker/       Periodic analytics workers
migrations/            PostgreSQL schema
web/templates/         Server-rendered HTML
web/static/            CSS
loadtest/              k6 scripts
docs/                  Engineering and resume notes
```

## Resume Angles

This rewrite is intentionally structured around interview-friendly backend and data-engineering work:

- Go service rewrite from a Rails monolith.
- PostgreSQL schema design with composite indexes and foreign keys.
- Idempotent event ingestion and append-only analytics history.
- Batched SQL rollups with snapshotting.
- Prometheus-style observability.
- k6 load-test harness and Go benchmarks.
