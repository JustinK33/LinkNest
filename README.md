# LinkNest

[![CI](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml/badge.svg)](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml)

A link-in-bio app in Go and MySQL/TiDB where every click is recorded as an event and rolled up into analytics.

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
| Database | MySQL and TiDB Cloud Serverless, via `go-sql-driver/mysql` |
| Passwords | bcrypt from `golang.org/x/crypto` |
| Templates | `html/template`, compiled into the binary with `embed` |
| Metrics | A hand-rolled in-memory registry rendered as Prometheus text |
| Deploy | Docker Compose, Kamal, nginx and Certbot, or a single Vercel function |
| Load testing | k6 |

`go.mod` has exactly two non-indirect requires. Sessions, routing, CSRF, and the metrics format are standard library plus code in this repo.

## Architecture

```mermaid
flowchart TD
    visitor["Visitor browser"] -->|"GET /slug, POST /links/id/track_click"| server["http/server.go<br/>routing, signed sessions"]
    main["cmd/linknest/main.go"] --> server
    vercel["api/index.go<br/>one Vercel function"] --> server
    server --> app["app/app.go<br/>html/template, embedded by web/embed.go"]
    server --> store["store/store.go<br/>validation and every SQL statement"]
    store -->|"INSERT ... ON DUPLICATE KEY UPDATE id = id"| db[("click_events, links, users, sessions")]
    worker["worker/worker.go<br/>5 min and 30 min tickers"] -->|"INSERT ... SELECT ... GROUP BY"| rollups[("hourly_link_stats, daily_user_stats, analytics_snapshots")]
    main --> worker
    db --> worker
    server -->|"GET /metrics"| metrics["metrics/metrics.go<br/>Prometheus text format"]
```

Two entry points build the same app.
`cmd/linknest/main.go` is the long-lived binary: it starts the HTTP server and the worker manager, whose tickers fire the hourly aggregator every 5 minutes and the daily one every 30.
`api/index.go` is the Vercel path, and it wraps only the server, which is why the rollups don't run there. Pages, auth, links, and click tracking work fine on Vercel; the aggregate tables just stop advancing unless something long-lived is also pointed at the same database.

Every write goes through `store/store.go`, including validation, so no handler and no future non-form code path can write a row that skipped a check.
Both aggregators do their work in a single `INSERT ... SELECT ... GROUP BY ... ON DUPLICATE KEY UPDATE`, so the database aggregates in one pass rather than the app looping, and each run is bracketed by a `worker_runs` row that goes from `running` to `succeeded` with a `rows_processed` count.

## What building this taught me

**`sync.Once` around a database connection caches the failure too.** Deployed logs showed every request failing with "startup failed: context deadline exceeded". TiDB Cloud Serverless auto-pauses when idle, and waking it on a cold connection took longer than the 5 second ping timeout in `db.Open`. That part was a one-line fix. The real bug was that `api/index.go` used `sync.Once` to connect lazily, so a single slow wake-up poisoned the container permanently: every request after it kept failing until Vercel happened to recycle the instance. A mutex-guarded retry means a transient timeout stays transient. Serverless plus an auto-pausing database means cold start is the normal case, not the edge case.

**`internal/` is a compiler rule, and someone else's build shim can break it.** Vercel's Go runtime wraps the handler in a synthetic `main__vc__go__.go` compiled as a loose file in the `command-line-arguments` pseudo-package, and Go's internal-visibility rule rejects that regardless of where the importer actually lives. It built fine with `go build ./...` and failed only through Vercel. Every `internal/X` package moved to `X` at the repo root. This is an application binary rather than a library, so the privacy `internal/` was enforcing wasn't protecting anything from anyone.

**Validation at the edges is validation you can skip.** Email format, username charset, and every length were unchecked, and uniqueness was left entirely to the database, so a duplicate signup surfaced as a raw driver error and an over-long name would have surfaced as a MySQL 1406 or a silent truncation. It all moved into the store. The limits deliberately sit below the column widths so a long value produces a sentence the user can act on. The username pattern is also stricter than the slug generator, because the generator drops characters it doesn't recognise, which meant `"!!!"` was a valid signup with an empty public URL.

**`"//evil.example"` looks like a path and isn't one.** Link URLs are restricted to `http`, `https`, `mailto`, `tel`, and root-relative paths, and a host is required, since `"https://"` on its own parses without error. Protocol-relative URLs are rejected by name because they read like a same-site path and leave the site.

**CSS bugs are invisible in exactly one colour scheme, so a test has to look.** Three failures in one branch were undetectable until something checked for them: an `@import` for fonts that every browser had silently dropped for the life of the file, an undefined `var(--space-14)` that collapsed a whole band on the landing page, and opaque hex colours outside the token blocks that were fine in light mode and unreadable in dark. There's now a test asserting every CSS variable is defined and another rejecting opaque colours outside `:root` and the media query, translucent `rgba()` allowed because it composites over whatever is beneath it. I mutation-tested that second one by injecting a hex and a white fill to confirm it actually fails.

**Rendering the templates without a server was worth building.** `preview/` renders the real templates with fake data and no database, which is how the dark scheme and the 375px layout got reviewed page by page. It needs `app.Funcs` and `app.PageNames` exported so it renders pages the same way the server does instead of keeping a second copy that drifts. The `?scroll`, `?probe`, and `/frame` handles exist because headless Chrome screenshots the window rather than the document, and clamps its own window to 500px wide on macOS.

## Documentation

- [docs/DESIGN.md](docs/DESIGN.md) covers the write path in more depth: the event log, the index-per-query-path list, the k6 numbers (28,835 tracking requests at ~958/sec, 21.9 ms p95, no duplicate keys leaked), and the known gap that ingestion has no queue in front of it. Its SQL snippets still show the Postgres `ON CONFLICT` spelling from before the TiDB port; the shipped statements use `ON DUPLICATE KEY UPDATE`.
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

To deploy on Vercel, `api/index.go` wraps the app as a single function and `vercel.json` routes everything to it. Set `DATABASE_URL` (with `?tls=true` for TiDB Cloud) and `SESSION_SECRET` in the dashboard.
Run `cmd/linknest` somewhere long-lived if you want the rollups, since a serverless function isn't around long enough for a background ticker.
