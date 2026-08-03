# LinkNest

[![CI](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml/badge.svg)](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml)

A link-in-bio app in Go and MySQL/TiDB where every click gets tracked and rolled up into analytics.

<!-- TODO: add an architecture diagram here -->

## What It Does

A user signs up, adds their links, and gets a public page at their own slug - the classic "link in bio" page you'd put in a social media profile.
Every visit to a link is recorded as a click, and those clicks roll up into hourly and daily analytics on the user's dashboard.
It's built as a small, real service where the interesting parts are in the data layer: idempotent click ingestion, an append-only event log, and batched SQL rollups.

## Tech Stack

- Go
- MySQL / TiDB (`go-sql-driver/mysql`)
- Prometheus metrics
- Docker Compose
- k6 (load testing)

## Install and Run

```sh
docker compose up --build
```

Open `http://localhost:8081`. Compose starts MySQL and the web service; the app applies its migrations on boot.

Config comes from environment variables (never commit a `.env`):

```sh
ADDR=:8080
DATABASE_URL=mysql://linknest:linknest@localhost:3306/linknest
SESSION_SECRET=replace-with-a-long-random-secret
```

### Deploying to Vercel

`api/index.go` wraps the app as a single serverless function; `vercel.json` routes every request there. Set these env vars in the Vercel dashboard:

```sh
DATABASE_URL=mysql://user:password@host:4000/dbname?tls=true
SESSION_SECRET=replace-with-a-long-random-secret
```

The hourly/daily analytics rollups (`worker/`) don't run on Vercel - serverless functions aren't long-lived enough for a background ticker. Pages, auth, links, and click tracking all work; run `cmd/linknest` somewhere long-lived (Docker/Kamal) if you need the rollups too.
