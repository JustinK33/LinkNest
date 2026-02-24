# LinkNest

A scalable multi-tenant link-in-bio platform that allows users to create customizable public profile pages and track real-time analytics through optimized, write-heavy event logging.
[![CI](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml/badge.svg)](https://github.com/JustinK33/LinkNest/actions/workflows/ci.yml)

---

## Tech Stack

- Ruby
- Ruby on Rails
- MySQL
- Redis
- Sidekiq
- Devise
- Docker

---

## Features (updating...)

- **Multi-Tenant Architecture**
  - Users can create multiple profile pages
  - Slug-based routing (yourapp.com/justin)
  - Strict ownership scoping for secure data isolation

- **Optimized Read-Heavy Endpoints**
  - Indexed slug lookups
  - ETag / Last-Modified caching headers
  - Cache invalidation tied to profile updates

---

## 📚 What I Learned From This Project

- **Designing a Multi-Tenant System**
 Structured data models to isolate user data safely while supporting scalable public traffic across thousands of profiles.
   
- **Aggressive MySQL Indexing**
  Implemented compound indexes (profile_id, occurred_at) and unique constraints to maintain performance under scale.

- **Integrated Continuous Integration**
  Implemented GitHub action workflows for continuous integration and continuous delivery
   
- **Background Processing with Sidekiq**
  Built asynchronous aggregation jobs to maintain responsive dashboards while handling high event volume.

---

## Running the Project

### To run the project locally, follow these steps:

  1. Clone the repo (git clone <url>)
  2. Install dependencies (bundle install)
  3. Setup database (rails db:create db:migrate)
  4. Start Redis (redis-server)
  5. Start Rails (rails server)
  6. Start Sidekiq (in another terminal) (bundle exec sidekiq)
     
### Run with Docker

  1. Build image (docker build -t linknest:latest .)
  2. Run containers (docker compose up --build)
