package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"linknest/internal/auth"
	"linknest/internal/models"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) CreateUser(ctx context.Context, email string, password string, username string, first string, last string) (models.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return models.User{}, err
	}
	slug := auth.Slug(username)
	if slug == "" {
		return models.User{}, errors.New("username must contain letters or numbers")
	}
	var user models.User
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, username, slug, first_name, last_name, profile_color)
		VALUES ($1, $2, $3, $4, $5, $6, '#56738c')
		RETURNING id, email, username, slug, first_name, last_name, bio, profile_color
	`, email, hash, username, slug, first, last).Scan(
		&user.ID, &user.Email, &user.Username, &user.Slug, &user.FirstName, &user.LastName, &user.Bio, &user.ProfileColor,
	)
	return user, err
}

func (s *Store) Authenticate(ctx context.Context, email string, password string) (models.User, error) {
	var user models.User
	var hash string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, username, slug, first_name, last_name, bio, profile_color
		FROM users
		WHERE lower(email) = lower($1)
	`, email).Scan(&user.ID, &user.Email, &hash, &user.Username, &user.Slug, &user.FirstName, &user.LastName, &user.Bio, &user.ProfileColor)
	if err != nil {
		return models.User{}, err
	}
	if !auth.CheckPassword(hash, password) {
		return models.User{}, sql.ErrNoRows
	}
	return user, nil
}

func (s *Store) CreateSession(ctx context.Context, userID int64, ip string, ua string) (string, error) {
	sessionID, err := auth.NewSessionID()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, sessionID, userID, ip, ua, time.Now().Add(30*24*time.Hour))
	return sessionID, err
}

func (s *Store) UserBySession(ctx context.Context, sessionID string) (models.User, error) {
	var user models.User
	err := s.db.QueryRowContext(ctx, `
		SELECT users.id, users.email, users.username, users.slug, users.first_name, users.last_name, users.bio, users.profile_color
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.id = $1 AND sessions.expires_at > now()
	`, sessionID).Scan(&user.ID, &user.Email, &user.Username, &user.Slug, &user.FirstName, &user.LastName, &user.Bio, &user.ProfileColor)
	return user, err
}

func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

func (s *Store) UserBySlug(ctx context.Context, slug string) (models.User, error) {
	var user models.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, username, slug, first_name, last_name, bio, profile_color
		FROM users
		WHERE slug = $1
	`, slug).Scan(&user.ID, &user.Email, &user.Username, &user.Slug, &user.FirstName, &user.LastName, &user.Bio, &user.ProfileColor)
	return user, err
}

func (s *Store) UpdateProfile(ctx context.Context, userID int64, first string, last string, bio string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET first_name = $2, last_name = $3, bio = $4, updated_at = now()
		WHERE id = $1
	`, userID, first, last, bio)
	return err
}

func (s *Store) CreateLink(ctx context.Context, userID int64, title string, url string, public bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var position int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM links WHERE user_id = $1`, userID).Scan(&position); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO links (user_id, title, url, position, public)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, title, url, position, public); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) LinksForUser(ctx context.Context, userID int64, onlyPublic bool, limit int, afterID int64) ([]models.Link, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `
		SELECT id, user_id, title, url, position, public, icon_color, click_count
		FROM links
		WHERE user_id = $1 AND id > $2
	`
	args := []any{userID, afterID}
	if onlyPublic {
		query += ` AND public = true`
	}
	query += ` ORDER BY position, id LIMIT $3`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var links []models.Link
	for rows.Next() {
		var link models.Link
		if err := rows.Scan(&link.ID, &link.UserID, &link.Title, &link.URL, &link.Position, &link.Public, &link.IconColor, &link.ClickCount); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

func (s *Store) LinkForTracking(ctx context.Context, linkID int64) (models.Link, error) {
	var link models.Link
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, url, position, public, icon_color, click_count
		FROM links
		WHERE id = $1 AND public = true
	`, linkID).Scan(&link.ID, &link.UserID, &link.Title, &link.URL, &link.Position, &link.Public, &link.IconColor, &link.ClickCount)
	return link, err
}

func (s *Store) IngestClick(ctx context.Context, event models.ClickEvent) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
		INSERT INTO click_events (
			idempotency_key, user_id, link_id, referrer, user_agent, ip_address,
			country_code, device_type, browser_name, event_time
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (idempotency_key) DO NOTHING
	`, event.IdempotencyKey, event.UserID, event.LinkID, event.Referrer, event.UserAgent, event.IPAddress, event.CountryCode, event.DeviceType, event.BrowserName, event.EventTime)
	if err != nil {
		return false, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if inserted == 1 {
		if _, err := tx.ExecContext(ctx, `UPDATE links SET click_count = click_count + 1 WHERE id = $1`, event.LinkID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return inserted == 1, nil
}

func (s *Store) AggregatePreviousHour(ctx context.Context) (int64, error) {
	hour := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Hour)
	return s.aggregateHour(ctx, hour)
}

func (s *Store) aggregateHour(ctx context.Context, hour time.Time) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO worker_runs (worker_name, status) VALUES ('hourly_aggregator', 'running')`); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO hourly_link_stats (user_id, link_id, hour, click_count, unique_visitors, updated_at)
		SELECT user_id, link_id, date_trunc('hour', event_time) AS hour, COUNT(*), COUNT(DISTINCT ip_address), now()
		FROM click_events
		WHERE event_time >= $1 AND event_time < $2
		GROUP BY user_id, link_id, date_trunc('hour', event_time)
		ON CONFLICT (link_id, hour)
		DO UPDATE SET click_count = EXCLUDED.click_count, unique_visitors = EXCLUDED.unique_visitors, updated_at = now()
	`, hour, hour.Add(time.Hour))
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	if _, err := tx.ExecContext(ctx, `
		UPDATE worker_runs
		SET status = 'succeeded', finished_at = now(), rows_processed = $1
		WHERE id = (SELECT max(id) FROM worker_runs WHERE worker_name = 'hourly_aggregator')
	`, rows); err != nil {
		return 0, err
	}
	return rows, tx.Commit()
}

func (s *Store) AggregateDay(ctx context.Context, date time.Time) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	if _, err := tx.ExecContext(ctx, `INSERT INTO worker_runs (worker_name, status) VALUES ('daily_aggregator', 'running')`); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `
		WITH daily AS (
			SELECT user_id, COUNT(*) AS total_clicks, COUNT(DISTINCT ip_address) AS unique_visitors, MAX(id) AS source_event_max_id
			FROM click_events
			WHERE event_time >= $1 AND event_time < $2
			GROUP BY user_id
		),
		top_links AS (
			SELECT DISTINCT ON (user_id) user_id, link_id, COUNT(*) AS link_clicks
			FROM click_events
			WHERE event_time >= $1 AND event_time < $2
			GROUP BY user_id, link_id
			ORDER BY user_id, COUNT(*) DESC, link_id
		)
		INSERT INTO daily_user_stats (user_id, date, total_clicks, unique_visitors, top_link_id, top_link_clicks, updated_at)
		SELECT daily.user_id, $3::date, daily.total_clicks, daily.unique_visitors, top_links.link_id, top_links.link_clicks, now()
		FROM daily
		LEFT JOIN top_links ON top_links.user_id = daily.user_id
		ON CONFLICT (user_id, date)
		DO UPDATE SET total_clicks = EXCLUDED.total_clicks,
			unique_visitors = EXCLUDED.unique_visitors,
			top_link_id = EXCLUDED.top_link_id,
			top_link_clicks = EXCLUDED.top_link_clicks,
			updated_at = now()
	`, start, end, start)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO analytics_snapshots (user_id, snapshot_date, total_clicks, unique_visitors, top_link_id, source_event_max_id)
		SELECT daily.user_id, $3::date, daily.total_clicks, daily.unique_visitors, top_links.link_id, daily.source_event_max_id
		FROM (
			SELECT user_id, COUNT(*) AS total_clicks, COUNT(DISTINCT ip_address) AS unique_visitors, MAX(id) AS source_event_max_id
			FROM click_events
			WHERE event_time >= $1 AND event_time < $2
			GROUP BY user_id
		) daily
		LEFT JOIN (
			SELECT DISTINCT ON (user_id) user_id, link_id, COUNT(*) AS link_clicks
			FROM click_events
			WHERE event_time >= $1 AND event_time < $2
			GROUP BY user_id, link_id
			ORDER BY user_id, COUNT(*) DESC, link_id
		) top_links ON top_links.user_id = daily.user_id
		ON CONFLICT (user_id, snapshot_date)
		DO UPDATE SET total_clicks = EXCLUDED.total_clicks,
			unique_visitors = EXCLUDED.unique_visitors,
			top_link_id = EXCLUDED.top_link_id,
			source_event_max_id = EXCLUDED.source_event_max_id
	`, start, end, start); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE worker_runs
		SET status = 'succeeded', finished_at = now(), rows_processed = $1
		WHERE id = (SELECT max(id) FROM worker_runs WHERE worker_name = 'daily_aggregator')
	`, rows); err != nil {
		return 0, err
	}
	return rows, tx.Commit()
}

func (s *Store) CleanupOldClicks(ctx context.Context, retentionDays int) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM click_events WHERE event_time < now() - ($1::int * interval '1 day')`, retentionDays)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (s *Store) Dashboard(ctx context.Context, user models.User) (models.Dashboard, error) {
	links, err := s.LinksForUser(ctx, user.ID, false, 100, 0)
	if err != nil {
		return models.Dashboard{}, err
	}
	var total, unique int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(DISTINCT ip_address) FROM click_events WHERE user_id = $1`, user.ID).Scan(&total, &unique); err != nil {
		return models.Dashboard{}, err
	}
	top, err := s.topLinks(ctx, user.ID)
	if err != nil {
		return models.Dashboard{}, err
	}
	daily, err := s.daily(ctx, user.ID)
	if err != nil {
		return models.Dashboard{}, err
	}
	return models.Dashboard{User: user, Links: links, TotalClicks: total, UniqueVisitors: unique, TopLinks: top, Daily: daily}, nil
}

func (s *Store) topLinks(ctx context.Context, userID int64) ([]models.Link, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, title, url, position, public, icon_color, click_count
		FROM links
		WHERE user_id = $1
		ORDER BY click_count DESC, id
		LIMIT 5
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var links []models.Link
	for rows.Next() {
		var link models.Link
		if err := rows.Scan(&link.ID, &link.UserID, &link.Title, &link.URL, &link.Position, &link.Public, &link.IconColor, &link.ClickCount); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

func (s *Store) daily(ctx context.Context, userID int64) ([]models.DailyStat, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT date, total_clicks, unique_visitors, COALESCE(top_link_id, 0), top_link_clicks
		FROM daily_user_stats
		WHERE user_id = $1
		ORDER BY date DESC
		LIMIT 14
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stats []models.DailyStat
	for rows.Next() {
		var stat models.DailyStat
		if err := rows.Scan(&stat.Date, &stat.TotalClicks, &stat.UniqueVisitors, &stat.TopLinkID, &stat.TopLinkClicks); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

func (s *Store) APIHistory(ctx context.Context, userID int64, limit int, afterID int64) ([]models.ClickEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, idempotency_key, user_id, link_id, referrer, user_agent, ip_address, country_code, device_type, browser_name, event_time
		FROM click_events
		WHERE user_id = $1 AND id > $2
		ORDER BY id
		LIMIT $3
	`, userID, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []models.ClickEvent
	for rows.Next() {
		var event models.ClickEvent
		if err := rows.Scan(&event.ID, &event.IdempotencyKey, &event.UserID, &event.LinkID, &event.Referrer, &event.UserAgent, &event.IPAddress, &event.CountryCode, &event.DeviceType, &event.BrowserName, &event.EventTime); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) ExplainIndexes(ctx context.Context) (string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE schemaname = 'public'
	`)
	var count int
	if err := row.Scan(&count); err != nil {
		return "", err
	}
	return fmt.Sprintf("Postgres schema has %d public indexes for auth, profile lookup, click ingestion, and analytics queries.", count), nil
}
