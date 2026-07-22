# LinkNest

[![CI](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml/badge.svg)](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml)

A link-in-bio app in Go and PostgreSQL.
Users register, build a public profile of links, and every click gets tracked as an event that feeds hourly and daily analytics.

I built it to have a small, real service where the interesting parts are in the data layer: idempotent ingestion, an append-only event log, batched SQL rollups, and connection-pool tuning.

## Stack

- Go 1.26, standard library `net/http` and `database/sql`
- PostgreSQL via the `pgx` driver
- Server-rendered HTML templates
- Prometheus-style metrics on `/metrics`
- Docker Compose for local dev, k6 for load testing

## How it works

One Go process serves the HTML pages and the JSON API and runs the background aggregation workers.
PostgreSQL holds users, signed sessions, links, and an append-only `click_events` table.
Workers periodically roll raw events up into hourly and daily stats with `INSERT ... SELECT ... ON CONFLICT DO UPDATE`, so reporting reads never scan the raw event stream.

### Routes

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/` | Landing page |
| GET/POST | `/register` | Account creation |
| GET/POST | `/login` | Session login |
| POST | `/logout` | End session |
| GET | `/dashboard` | Links + analytics (auth) |
| POST | `/profile`, `/links` | Update profile, add a link (auth) |
| GET | `/{slug}` | Public profile |
| POST | `/links/{id}/track_click` | Idempotent click ingestion |
| GET | `/api/v1/events?limit=50&after=0` | Cursor-paginated event history (auth) |
| GET | `/api/v1/status` | Database status summary (auth) |
| GET | `/metrics` | Prometheus metrics |
| GET | `/up` | Health check |

### Things worth pointing out

- **Idempotent clicks.** `track_click` honors an `Idempotency-Key` header and derives a deterministic key (link + IP + user agent + referrer + one-minute bucket) when the client doesn't send one. The DB enforces uniqueness and `ON CONFLICT DO NOTHING` drops duplicates.
- **Append-only events.** `click_events` is never mutated, so analytics can always be replayed. Daily snapshots keep a `source_event_max_id` high-water mark for auditing after a worker crash.
- **Batched rollups.** Aggregation happens in single SQL statements rather than app-side loops.
- **Composite indexes** cover each real query path - dashboard ordering, public listing, unique-visitor counts, top-link ranking.
- **Tuned pooling** via `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME_SECONDS`.

## Running it

```sh
docker compose up --build
```

Then open http://localhost:8081.
Compose starts PostgreSQL and the web service; the app applies `migrations/001_init_postgres.sql` on boot.

### Configuration

All config comes from environment variables. Never commit a `.env`.

```sh
ADDR=:8080
DATABASE_URL=postgres://linknest:linknest@localhost:5432/linknest?sslmode=disable
SESSION_SECRET=replace-with-a-long-random-secret
DB_MAX_OPEN_CONNS=20
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME_SECONDS=300
```

Compose reads `GO_DATABASE_URL` from your `.env` and passes it into the container as `DATABASE_URL`.

## Testing

```sh
go test ./... -bench=. -benchmem
```

Load test (after creating a user with a public link):

```sh
k6 run -e BASE_URL=http://localhost:8081 -e LINK_ID=1 -e VUS=100 -e DURATION=30s loadtest/k6-clicks.js
```

On my machine (Apple M4, Docker Compose stack) a 30s run at 100 VUs sustained ~958 clicks/sec at 21.9 ms p95 with zero failures, and the hourly/daily rollups each processed the ~28.8k resulting events in under 45 ms.

## Layout

```text
cmd/linknest/     entrypoint
internal/auth/    passwords, signed sessions, slugs, idempotency keys
internal/config/  env config
internal/db/      connection + migration runner
internal/http/    routes and handlers
internal/metrics/ in-memory Prometheus registry
internal/models/  domain structs
internal/store/   SQL, transactions, aggregation
internal/worker/  periodic analytics workers
migrations/       schema
web/              templates + CSS
loadtest/         k6 scripts
```
