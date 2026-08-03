CREATE TABLE IF NOT EXISTS users (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  username VARCHAR(255) NOT NULL,
  slug VARCHAR(255) NOT NULL,
  first_name VARCHAR(255) NOT NULL,
  last_name VARCHAR(255) NOT NULL,
  bio TEXT NOT NULL,
  profile_color VARCHAR(16) NOT NULL DEFAULT '#3b82f6',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY idx_users_email (email),
  UNIQUE KEY idx_users_username (username),
  UNIQUE KEY idx_users_slug (slug)
);

CREATE TABLE IF NOT EXISTS sessions (
  id VARCHAR(64) PRIMARY KEY,
  user_id BIGINT NOT NULL,
  ip_address VARCHAR(64) NOT NULL DEFAULT '',
  user_agent VARCHAR(512) NOT NULL DEFAULT '',
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_sessions_user_expires (user_id, expires_at)
);

CREATE TABLE IF NOT EXISTS links (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  title VARCHAR(255) NOT NULL,
  url TEXT NOT NULL,
  position INT NOT NULL DEFAULT 0,
  public BOOLEAN NOT NULL DEFAULT true,
  icon_color VARCHAR(16) NOT NULL DEFAULT '#3b82f6',
  click_count BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_links_user_position (user_id, position),
  KEY idx_links_user_public_position (user_id, public, position),
  KEY idx_links_user_click_count (user_id, click_count DESC)
);

CREATE TABLE IF NOT EXISTS click_events (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  idempotency_key VARCHAR(64) NOT NULL,
  user_id BIGINT NOT NULL,
  link_id BIGINT NOT NULL,
  referrer VARCHAR(1024) NOT NULL DEFAULT '',
  user_agent VARCHAR(512) NOT NULL DEFAULT '',
  ip_address VARCHAR(64) NOT NULL DEFAULT '',
  country_code VARCHAR(8) NOT NULL DEFAULT '',
  device_type VARCHAR(32) NOT NULL DEFAULT '',
  browser_name VARCHAR(32) NOT NULL DEFAULT '',
  event_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ingested_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY idx_click_events_idempotency (idempotency_key),
  KEY idx_click_events_user_time (user_id, event_time),
  KEY idx_click_events_link_time (link_id, event_time),
  KEY idx_click_events_time (event_time),
  KEY idx_click_events_link_time_ip (link_id, event_time, ip_address)
);

CREATE TABLE IF NOT EXISTS hourly_link_stats (
  user_id BIGINT NOT NULL,
  link_id BIGINT NOT NULL,
  hour DATETIME NOT NULL,
  click_count BIGINT NOT NULL DEFAULT 0,
  unique_visitors BIGINT NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (link_id, hour),
  KEY idx_hourly_stats_user_hour (user_id, hour),
  KEY idx_hourly_stats_hour (hour)
);

CREATE TABLE IF NOT EXISTS daily_user_stats (
  user_id BIGINT NOT NULL,
  date DATE NOT NULL,
  total_clicks BIGINT NOT NULL DEFAULT 0,
  unique_visitors BIGINT NOT NULL DEFAULT 0,
  top_link_id BIGINT,
  top_link_clicks BIGINT NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, date),
  KEY idx_daily_stats_date (date)
);

CREATE TABLE IF NOT EXISTS analytics_snapshots (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  snapshot_date DATE NOT NULL,
  total_clicks BIGINT NOT NULL DEFAULT 0,
  unique_visitors BIGINT NOT NULL DEFAULT 0,
  top_link_id BIGINT,
  source_event_max_id BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY idx_analytics_snapshots_user_date (user_id, snapshot_date)
);

CREATE TABLE IF NOT EXISTS worker_runs (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  worker_name VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  finished_at DATETIME NULL,
  rows_processed BIGINT NOT NULL DEFAULT 0,
  error VARCHAR(1024) NOT NULL DEFAULT '',
  KEY idx_worker_runs_name_started (worker_name, started_at DESC)
);
