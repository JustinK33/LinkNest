# LinkNest

[![CI](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml/badge.svg)](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml)

A link-in-bio app in Go and PostgreSQL where every click gets tracked and rolled up into analytics.

<!-- TODO: add an architecture diagram here -->

## What It Does

A user signs up, adds their links, and gets a public page at their own slug - the classic "link in bio" page you'd put in a social media profile.
Every visit to a link is recorded as a click, and those clicks roll up into hourly and daily analytics on the user's dashboard.
It's built as a small, real service where the interesting parts are in the data layer: idempotent click ingestion, an append-only event log, and batched SQL rollups.

## Tech Stack

- Go
- PostgreSQL (`pgx`)
- Prometheus metrics
- Docker Compose
- k6 (load testing)

## Install and Run

```sh
docker compose up --build
```

Open `http://localhost:8081`. Compose starts PostgreSQL and the web service; the app applies its migrations on boot.

Config comes from environment variables (never commit a `.env`):

```sh
ADDR=:8080
DATABASE_URL=postgres://linknest:linknest@localhost:5432/linknest?sslmode=disable
SESSION_SECRET=replace-with-a-long-random-secret
```
