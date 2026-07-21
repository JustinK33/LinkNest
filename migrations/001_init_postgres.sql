CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  username TEXT NOT NULL UNIQUE,
  slug TEXT NOT NULL UNIQUE,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  profile_color TEXT NOT NULL DEFAULT '#3b82f6',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  ip_address TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_expires ON sessions(user_id, expires_at);

CREATE TABLE IF NOT EXISTS links (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  url TEXT NOT NULL,
  position INTEGER NOT NULL DEFAULT 0,
  public BOOLEAN NOT NULL DEFAULT true,
  icon_color TEXT NOT NULL DEFAULT '#3b82f6',
  click_count BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_links_user_position ON links(user_id, position);
CREATE INDEX IF NOT EXISTS idx_links_user_public_position ON links(user_id, public, position);
CREATE INDEX IF NOT EXISTS idx_links_user_click_count ON links(user_id, click_count DESC);

CREATE TABLE IF NOT EXISTS click_events (
  id BIGSERIAL PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  link_id BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  referrer TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  ip_address TEXT NOT NULL DEFAULT '',
  country_code TEXT NOT NULL DEFAULT '',
  device_type TEXT NOT NULL DEFAULT '',
  browser_name TEXT NOT NULL DEFAULT '',
  event_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  ingested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_click_events_user_time ON click_events(user_id, event_time);
CREATE INDEX IF NOT EXISTS idx_click_events_link_time ON click_events(link_id, event_time);
CREATE INDEX IF NOT EXISTS idx_click_events_time ON click_events(event_time);
CREATE INDEX IF NOT EXISTS idx_click_events_link_time_ip ON click_events(link_id, event_time, ip_address);

CREATE TABLE IF NOT EXISTS hourly_link_stats (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  link_id BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  hour TIMESTAMPTZ NOT NULL,
  click_count BIGINT NOT NULL DEFAULT 0,
  unique_visitors BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (link_id, hour)
);

CREATE INDEX IF NOT EXISTS idx_hourly_stats_user_hour ON hourly_link_stats(user_id, hour);
CREATE INDEX IF NOT EXISTS idx_hourly_stats_hour ON hourly_link_stats(hour);

CREATE TABLE IF NOT EXISTS daily_user_stats (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  total_clicks BIGINT NOT NULL DEFAULT 0,
  unique_visitors BIGINT NOT NULL DEFAULT 0,
  top_link_id BIGINT REFERENCES links(id) ON DELETE SET NULL,
  top_link_clicks BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_user_stats(date);

CREATE TABLE IF NOT EXISTS analytics_snapshots (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  snapshot_date DATE NOT NULL,
  total_clicks BIGINT NOT NULL DEFAULT 0,
  unique_visitors BIGINT NOT NULL DEFAULT 0,
  top_link_id BIGINT REFERENCES links(id) ON DELETE SET NULL,
  source_event_max_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, snapshot_date)
);

CREATE TABLE IF NOT EXISTS worker_runs (
  id BIGSERIAL PRIMARY KEY,
  worker_name TEXT NOT NULL,
  status TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at TIMESTAMPTZ,
  rows_processed BIGINT NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_worker_runs_name_started ON worker_runs(worker_name, started_at DESC);
