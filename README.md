# LinkNest

[![CI](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml/badge.svg)](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml)

A link-in-bio app in Go and MySQL where every click is recorded as an event and rolled up into analytics.

## What it does

A user signs up, adds their links, and gets a public page at their own slug, the ordinary link-in-bio page you'd put in a social profile.
The product surface is deliberately small. The interesting parts are underneath it: click ingestion is idempotent, the event log is append-only, and the analytics are batched SQL rollups derived from that log rather than counters that get incremented and hoped over.

`POST /links/{id}/track_click` is called from the browser and is safe to retry.
It takes an `Idempotency-Key` header, and when the client doesn't send one the server derives a deterministic key from link id, IP, user agent, referrer, and a one-minute time bucket.
That column is unique and the insert is a MySQL `ON DUPLICATE KEY UPDATE id = id`, so a double-fired or retried click collapses to one row.
The event insert and the `links.click_count` increment share a transaction, so the cached counter cannot drift from the log it summarizes.

Because `click_events` is never updated in place, a wrong rollup is recoverable: `analytics_snapshots` stores each day's totals next to `source_event_max_id`, the highest event id that fed it, so after a worker crash you compare that watermark against the live event stream instead of re-scanning everything.

The whole thing has two direct dependencies. Everything else is the standard library.

## Tech stack

| Layer | What it uses |
| --- | --- |
| Language | Go 1.26 |
| Database | MySQL |
| Passwords | bcrypt from `golang.org/x/crypto` |
| Templates | `html/template`, compiled into the binary with `embed` |
| Metrics | A hand-rolled in-memory registry rendered as Prometheus text |
| Deploy | A GCP VM running the container behind nginx, with Certbot for TLS |
| Load testing | k6 |

`go.mod` has exactly two non-indirect requires. Sessions, routing, CSRF, and the metrics format are standard library plus code in this repo.

## Architecture

```mermaid
flowchart TD
    visitor["Visitor browser"] -->|"GET /slug, POST /links/id/track_click"| nginx["nginx on the GCP VM<br/>TLS from Certbot, static caching"]
    nginx -->|"proxy_pass to :8080"| server["http/server.go<br/>routing, signed sessions"]
    main["cmd/linknest/main.go"] --> server
    server --> app["app/app.go<br/>html/template, embedded by web/embed.go"]
    server --> store["store/store.go<br/>validation and every SQL statement"]
    store -->|"INSERT ... ON DUPLICATE KEY UPDATE id = id"| db[("click_events, links, users, sessions")]
    worker["worker/worker.go<br/>5 min and 30 min tickers"] -->|"INSERT ... SELECT ... GROUP BY"| rollups[("hourly_link_stats, daily_user_stats, analytics_snapshots")]
    main --> worker
    db --> worker
    server -->|"GET /metrics"| metrics["metrics/metrics.go<br/>Prometheus text format"]
```

It runs on a GCP VM: the container from `docker-compose.yaml` listening on 8080, nginx in front of it terminating TLS and caching static assets, and Certbot renewing the certificate for `linknest.info`.
`cmd/linknest/main.go` is the entry point that deployment uses, and it starts both halves of the app: the HTTP server, and the worker manager whose tickers fire the hourly aggregator every 5 minutes and the daily one every 30.

Every write goes through `store/store.go`, including validation, so no handler and no future non-form code path can write a row that skipped a check.
Both aggregators do their work in a single `INSERT ... SELECT ... GROUP BY ... ON DUPLICATE KEY UPDATE`, so the database aggregates in one pass rather than the app looping, and each run is bracketed by a `worker_runs` row that goes from `running` to `succeeded` with a `rows_processed` count.

## What building this taught me

**CSS bugs are invisible in exactly one colour scheme, so a test has to look.**
One branch hid three failures nothing could see: a font `@import` every browser had silently dropped, an undefined `var(--space-14)` collapsing a whole landing-page band, and opaque hex colours that were fine in light mode and unreadable in dark.
There are now tests asserting every CSS variable is defined and rejecting opaque colours outside the token blocks, and I mutation-tested the second one to confirm it actually fails.

## Documentation

- [docs/DESIGN.md](docs/DESIGN.md) covers the write path in more depth: the event log, the index-per-query-path list, the k6 numbers (28,835 tracking requests at ~958/sec, 21.9 ms p95, no duplicate keys leaked), and the known gap that ingestion has no queue in front of it. Its SQL snippets still show the Postgres `ON CONFLICT` spelling from before the MySQL port; the shipped statements use `ON DUPLICATE KEY UPDATE`.
- [db/migrations/001_init_mysql.sql](db/migrations/001_init_mysql.sql) is the schema, applied on boot.
- [loadtest/k6-clicks.js](loadtest/k6-clicks.js) is the ingestion load test the measured numbers come from.

## Quick start

```sh
docker compose up --build
```

Open `http://localhost:8081`. Compose brings up MySQL and the web service, and the app applies its own migrations on boot.

Config is environment variables only, and `.env` is gitignored:

```sh
ADDR=:8080
DATABASE_URL=mysql://linknest:linknest@localhost:3306/linknest
SESSION_SECRET=replace-with-a-long-random-secret
```

Tests and the build are what CI runs:

```sh
go test ./...
go build ./cmd/linknest
```

Deployment is the same Compose stack, the app and MySQL, on a GCP VM.
nginx sits in front of it: [ops/nginx/linknest.info.conf](ops/nginx/linknest.info.conf) is the server block, [ops/nginx/docker/init-letsencrypt.sh](ops/nginx/docker/init-letsencrypt.sh) issues the first certificate, and [ops/nginx/certbot-renew-hook.sh](ops/nginx/certbot-renew-hook.sh) reloads nginx after a renewal.
Running the whole binary is what keeps the rollups advancing, since a background ticker needs a process that stays up.

## License

MIT - see [LICENSE](LICENSE).
