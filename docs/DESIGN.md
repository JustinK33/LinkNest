# Design notes

Why the pieces of LinkNest are built the way they are.
The product is small on purpose; most of the effort went into the write path and the analytics layer.

## Events as the source of truth

Every click is appended to `click_events` and never updated in place.
The aggregate tables (`hourly_link_stats`, `daily_user_stats`) are derived views of that log, so if a rollup is ever wrong it can be recomputed from the raw events.

`analytics_snapshots` stores each day's totals alongside `source_event_max_id`, the highest event id that fed the snapshot.
After a worker crash you can compare that watermark against the live event stream to see exactly what still needs processing, without re-scanning everything.

## Idempotent ingestion

`POST /links/{id}/track_click` is meant to be called from the browser and retried freely.
It accepts an `Idempotency-Key` header; when the client doesn't send one, the server derives a deterministic key from link id, IP, user agent, referrer, and a one-minute time bucket.
`click_events.idempotency_key` is unique, and the insert uses `ON CONFLICT DO NOTHING`, so a retried or double-fired click collapses to a single row.

The event insert and the `links.click_count` increment run in one transaction, so the cached counter can't drift from the log.

## Schema and indexes

Indexes follow the actual query paths rather than being added speculatively:

- `idx_links_user_position` - dashboard ordering
- `idx_links_user_public_position` - public profile rendering
- `idx_links_user_click_count` - top-link ranking
- `idx_click_events_user_time` / `idx_click_events_link_time` - analytics windows
- `idx_click_events_link_time_ip` - unique-visitor counts
- `idx_hourly_stats_user_hour` - dashboard rollups

## Batched rollups

Both aggregators do the work in a single `INSERT ... SELECT ... GROUP BY ... ON CONFLICT DO UPDATE` statement so PostgreSQL aggregates in one pass instead of the app looping row by row.
Each run is bracketed by a `worker_runs` row (`running` -> `succeeded` with `rows_processed`) for a basic audit trail.
The daily worker also trims events older than 90 days.

## Observability

The `/metrics` endpoint renders a Prometheus-compatible text format from an in-memory registry: request count and latency, clicks ingested vs. deduplicated, worker run counts, durations, rows processed, and retention deletes.

## Security

- Passwords hashed with bcrypt.
- Session cookies signed with HMAC-SHA256, `HttpOnly`, `SameSite=Lax`.
- All secrets come from the environment; `.env` is gitignored.
- Click tracking only accepts public links.
- Submitted URLs are normalized to `https://` and restricted to `http`, `https`, `mailto`, `tel`, and relative paths.

## Measured behavior

k6, 100 VUs for 30s against the Docker Compose stack on an Apple M4:

- 28,835 tracking requests, ~958/sec, 21.9 ms p95, 0% failures
- 28,835 rows persisted, 28,835 unique idempotency keys (no duplicates leaked)
- hourly rollup processed the batch in 36.4 ms, daily in 40.9 ms

Microbenchmarks: `IdempotencyKey` ~229 ns/op, `MetricsIncrement` ~13.8 ns/op with zero allocations.

## Known gaps

Ingestion writes straight to PostgreSQL with no queue in front of it.
Under sustained burst traffic the natural next step is a bounded in-process channel (or an external queue) between the HTTP handler and the DB, with a backpressure signal when it fills.
The metrics registry can already expose queue depth once that exists.
