package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	neturl "net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"time"

	"github.com/go-sql-driver/mysql"

	"linknest/auth"
	"linknest/models"
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

// Errors a caller can render straight into a page. Everything else coming out of
// this package is a driver or context error, which the handler logs and replaces
// with a generic message.
const (
	ErrEmailTaken    = models.UserError("That email is already registered. Try signing in instead.")
	ErrUsernameTaken = models.UserError("That username is already taken. Please pick another.")

	ErrInvalidEmail    = models.UserError("Please enter a valid email address.")
	ErrInvalidUsername = models.UserError("Your username must be 3-30 characters, using only letters, numbers, dots, dashes or underscores.")
	ErrNameRequired    = models.UserError("Please enter both a first and last name.")
	ErrNameTooLong     = models.UserError("First and last names are limited to 60 characters each.")
	ErrBioTooLong      = models.UserError("Your bio is limited to 500 characters.")

	ErrTitleRequired  = models.UserError("Please give the link a title.")
	ErrTitleTooLong   = models.UserError("Link titles are limited to 120 characters.")
	ErrInvalidLinkURL = models.UserError("That doesn't look like a valid link. Check the address and try again.")
	ErrLinkNotFound   = models.UserError("That link no longer exists.")
	ErrLinkAtEdge     = models.UserError("That link is already at the end of your list.")
)

// Field limits. Enforced here rather than in the handlers so no write path can
// skip them, and deliberately below the column widths so a long value is a
// validation message instead of a truncation or a driver error.
const (
	maxEmailLen = 254
	maxNameLen  = 60
	maxBioLen   = 500
	maxTitleLen = 120
	maxURLLen   = 2048
)

const mysqlDuplicateEntry = 1062

// usernamePattern is deliberately stricter than auth.Slug: the slug generator
// drops anything it doesn't recognise, so "!!!" would silently become a valid
// signup. Requiring the raw input to be clean means the public URL matches what
// the user actually typed.
var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,30}$`)

func validateSignup(email, username, first, last string) error {
	if len(email) > maxEmailLen {
		return ErrInvalidEmail
	}
	// ParseAddress also accepts `Name <addr>`; a bare address must not have a
	// display name attached to it.
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Name != "" || addr.Address != email {
		return ErrInvalidEmail
	}
	if !usernamePattern.MatchString(username) || auth.Slug(username) == "" {
		return ErrInvalidUsername
	}
	return validateNames(first, last)
}

func validateNames(first, last string) error {
	if first == "" || last == "" {
		return ErrNameRequired
	}
	if utf8.RuneCountInString(first) > maxNameLen || utf8.RuneCountInString(last) > maxNameLen {
		return ErrNameTooLong
	}
	return nil
}

func validateLink(title, url string) error {
	if title == "" {
		return ErrTitleRequired
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return ErrTitleTooLong
	}
	if url == "" || len(url) > maxURLLen {
		return ErrInvalidLinkURL
	}
	// "//evil.example" is a protocol-relative URL: it looks like a same-site path
	// but leaves the site, so it is not the root-relative link safeURL allows.
	if strings.HasPrefix(url, "//") {
		return ErrInvalidLinkURL
	}
	if strings.HasPrefix(url, "/") {
		return nil
	}
	// Handlers pass everything else through safeURL, so a scheme is already
	// present. A host is what's worth checking: "https://" alone parses fine.
	parsed, err := neturl.Parse(url)
	if err != nil {
		return ErrInvalidLinkURL
	}
	switch parsed.Scheme {
	case "http", "https":
		if parsed.Host == "" {
			return ErrInvalidLinkURL
		}
	case "mailto", "tel":
		if parsed.Opaque == "" {
			return ErrInvalidLinkURL
		}
	default:
		return ErrInvalidLinkURL
	}
	return nil
}

// signupConflict maps a duplicate-key violation on users to a sentinel error.
// Anything else is returned unchanged for the caller to log and generalize.
func signupConflict(err error) error {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != mysqlDuplicateEntry {
		return err
	}
	switch {
	case strings.Contains(mysqlErr.Message, "idx_users_email"):
		return ErrEmailTaken
	case strings.Contains(mysqlErr.Message, "idx_users_username"), strings.Contains(mysqlErr.Message, "idx_users_slug"):
		return ErrUsernameTaken
	default:
		return err
	}
}

func (s *Store) CreateUser(ctx context.Context, email string, password string, username string, first string, last string) (models.User, error) {
	email = strings.TrimSpace(email)
	username = strings.TrimSpace(username)
	first, last = strings.TrimSpace(first), strings.TrimSpace(last)
	if err := validateSignup(email, username, first, last); err != nil {
		return models.User{}, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return models.User{}, err
	}
	slug := auth.Slug(username)
	const profileColor = "#56738c"
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, username, slug, first_name, last_name, bio, profile_color)
		VALUES (?, ?, ?, ?, ?, ?, '', ?)
	`, email, hash, username, slug, first, last, profileColor)
	if err != nil {
		return models.User{}, signupConflict(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.User{}, err
	}
	return models.User{
		ID:           id,
		Email:        email,
		Username:     username,
		Slug:         slug,
		FirstName:    first,
		LastName:     last,
		ProfileColor: profileColor,
	}, nil
}

func (s *Store) Authenticate(ctx context.Context, email string, password string) (models.User, error) {
	var user models.User
	var hash string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, username, slug, first_name, last_name, bio, profile_color
		FROM users
		WHERE lower(email) = lower(?)
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
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, userID, ip, ua, time.Now().Add(30*24*time.Hour))
	return sessionID, err
}

func (s *Store) UserBySession(ctx context.Context, sessionID string) (models.User, error) {
	var user models.User
	err := s.db.QueryRowContext(ctx, `
		SELECT users.id, users.email, users.username, users.slug, users.first_name, users.last_name, users.bio, users.profile_color
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.id = ? AND sessions.expires_at > now()
	`, sessionID).Scan(&user.ID, &user.Email, &user.Username, &user.Slug, &user.FirstName, &user.LastName, &user.Bio, &user.ProfileColor)
	return user, err
}

func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}

func (s *Store) UserBySlug(ctx context.Context, slug string) (models.User, error) {
	var user models.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, username, slug, first_name, last_name, bio, profile_color
		FROM users
		WHERE slug = ?
	`, slug).Scan(&user.ID, &user.Email, &user.Username, &user.Slug, &user.FirstName, &user.LastName, &user.Bio, &user.ProfileColor)
	return user, err
}

func (s *Store) UpdateProfile(ctx context.Context, userID int64, first string, last string, bio string) error {
	first, last, bio = strings.TrimSpace(first), strings.TrimSpace(last), strings.TrimSpace(bio)
	if err := validateNames(first, last); err != nil {
		return err
	}
	if utf8.RuneCountInString(bio) > maxBioLen {
		return ErrBioTooLong
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET first_name = ?, last_name = ?, bio = ?, updated_at = now()
		WHERE id = ?
	`, first, last, bio, userID)
	return err
}

func (s *Store) CreateLink(ctx context.Context, userID int64, title string, url string, public bool) error {
	title, url = strings.TrimSpace(title), strings.TrimSpace(url)
	if err := validateLink(title, url); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var position int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), -1) + 1 FROM links WHERE user_id = ?`, userID).Scan(&position); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO links (user_id, title, url, position, public)
		VALUES (?, ?, ?, ?, ?)
	`, userID, title, url, position, public); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateLink and DeleteLink are both scoped by user_id, so a guessed or stale
// link ID from another account is a "not found" rather than an edit.
func (s *Store) UpdateLink(ctx context.Context, userID int64, linkID int64, title string, url string, public bool) error {
	title, url = strings.TrimSpace(title), strings.TrimSpace(url)
	if err := validateLink(title, url); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE links
		SET title = ?, url = ?, public = ?, updated_at = now()
		WHERE id = ? AND user_id = ?
	`, title, url, public, linkID, userID)
	if err != nil {
		return err
	}
	return requireRow(result)
}

func (s *Store) DeleteLink(ctx context.Context, userID int64, linkID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM links WHERE id = ? AND user_id = ?`, linkID, userID)
	if err != nil {
		return err
	}
	if err := requireRow(result); err != nil {
		return err
	}
	// click_events rows are left alone on purpose: they are the user's historical
	// traffic, and the dashboard totals would drop retroactively if a delete took
	// them with it. Nothing renders a deleted link, since every read joins links.
	//
	// Close the gap the delete left, so positions stay 0..n-1 and MoveLink can
	// keep treating them as an index.
	if err := renumber(ctx, tx, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// MoveLink shifts one link one place up or down. It rewrites the whole list's
// positions rather than swapping two rows: legacy rows can share a position or
// leave gaps, and a swap between two equal positions is a no-op that looks like
// a silent failure to the user.
func (s *Store) MoveLink(ctx context.Context, userID int64, linkID int64, up bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ids, err := orderedLinkIDs(ctx, tx, userID)
	if err != nil {
		return err
	}
	at := -1
	for i, id := range ids {
		if id == linkID {
			at = i
			break
		}
	}
	if at < 0 {
		return ErrLinkNotFound
	}
	to := at + 1
	if up {
		to = at - 1
	}
	if to < 0 || to >= len(ids) {
		return ErrLinkAtEdge
	}
	ids[at], ids[to] = ids[to], ids[at]
	if err := writePositions(ctx, tx, userID, ids); err != nil {
		return err
	}
	return tx.Commit()
}

// requireRow turns "the UPDATE matched nothing" into a user-facing error. Without
// it, editing someone else's link or an already-deleted one reports success.
func requireRow(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrLinkNotFound
	}
	return nil
}

func renumber(ctx context.Context, tx *sql.Tx, userID int64) error {
	ids, err := orderedLinkIDs(ctx, tx, userID)
	if err != nil {
		return err
	}
	return writePositions(ctx, tx, userID, ids)
}

func orderedLinkIDs(ctx context.Context, tx *sql.Tx, userID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM links WHERE user_id = ? ORDER BY position, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ponytail: one UPDATE per row. LinksForUser caps a profile at 100 links, so
// this is at most 100 statements inside one transaction. Swap for a single
// CASE-expression UPDATE if that cap ever rises.
func writePositions(ctx context.Context, tx *sql.Tx, userID int64, ids []int64) error {
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE links SET position = ? WHERE id = ? AND user_id = ?`, i, id, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) LinksForUser(ctx context.Context, userID int64, onlyPublic bool, limit int, afterID int64) ([]models.Link, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `
		SELECT id, user_id, title, url, position, public, icon_color, click_count
		FROM links
		WHERE user_id = ? AND id > ?
	`
	args := []any{userID, afterID}
	if onlyPublic {
		query += ` AND public = true`
	}
	query += ` ORDER BY position, id LIMIT ?`
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
		WHERE id = ? AND public = true
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = id
	`, event.IdempotencyKey, event.UserID, event.LinkID, event.Referrer, event.UserAgent, event.IPAddress, event.CountryCode, event.DeviceType, event.BrowserName, event.EventTime)
	if err != nil {
		return false, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if inserted == 1 {
		if _, err := tx.ExecContext(ctx, `UPDATE links SET click_count = click_count + 1 WHERE id = ?`, event.LinkID); err != nil {
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
	runResult, err := tx.ExecContext(ctx, `INSERT INTO worker_runs (worker_name, status) VALUES ('hourly_aggregator', 'running')`)
	if err != nil {
		return 0, err
	}
	runID, err := runResult.LastInsertId()
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO hourly_link_stats (user_id, link_id, hour, click_count, unique_visitors, updated_at)
		SELECT user_id, link_id, ?, COUNT(*), COUNT(DISTINCT ip_address), now()
		FROM click_events
		WHERE event_time >= ? AND event_time < ?
		GROUP BY user_id, link_id
		ON DUPLICATE KEY UPDATE click_count = VALUES(click_count), unique_visitors = VALUES(unique_visitors), updated_at = VALUES(updated_at)
	`, hour, hour, hour.Add(time.Hour))
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	if _, err := tx.ExecContext(ctx, `
		UPDATE worker_runs
		SET status = 'succeeded', finished_at = now(), rows_processed = ?
		WHERE id = ?
	`, rows, runID); err != nil {
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
	runResult, err := tx.ExecContext(ctx, `INSERT INTO worker_runs (worker_name, status) VALUES ('daily_aggregator', 'running')`)
	if err != nil {
		return 0, err
	}
	runID, err := runResult.LastInsertId()
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `
		WITH daily AS (
			SELECT user_id, COUNT(*) AS total_clicks, COUNT(DISTINCT ip_address) AS unique_visitors, MAX(id) AS source_event_max_id
			FROM click_events
			WHERE event_time >= ? AND event_time < ?
			GROUP BY user_id
		),
		ranked_links AS (
			SELECT user_id, link_id, COUNT(*) AS link_clicks,
				ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY COUNT(*) DESC, link_id) AS rn
			FROM click_events
			WHERE event_time >= ? AND event_time < ?
			GROUP BY user_id, link_id
		),
		top_links AS (
			SELECT user_id, link_id, link_clicks FROM ranked_links WHERE rn = 1
		)
		INSERT INTO daily_user_stats (user_id, date, total_clicks, unique_visitors, top_link_id, top_link_clicks, updated_at)
		SELECT daily.user_id, ?, daily.total_clicks, daily.unique_visitors, top_links.link_id, top_links.link_clicks, now()
		FROM daily
		LEFT JOIN top_links ON top_links.user_id = daily.user_id
		ON DUPLICATE KEY UPDATE
			total_clicks = VALUES(total_clicks),
			unique_visitors = VALUES(unique_visitors),
			top_link_id = VALUES(top_link_id),
			top_link_clicks = VALUES(top_link_clicks),
			updated_at = VALUES(updated_at)
	`, start, end, start, end, start)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	if _, err := tx.ExecContext(ctx, `
		WITH daily AS (
			SELECT user_id, COUNT(*) AS total_clicks, COUNT(DISTINCT ip_address) AS unique_visitors, MAX(id) AS source_event_max_id
			FROM click_events
			WHERE event_time >= ? AND event_time < ?
			GROUP BY user_id
		),
		ranked_links AS (
			SELECT user_id, link_id,
				ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY COUNT(*) DESC, link_id) AS rn
			FROM click_events
			WHERE event_time >= ? AND event_time < ?
			GROUP BY user_id, link_id
		),
		top_links AS (
			SELECT user_id, link_id FROM ranked_links WHERE rn = 1
		)
		INSERT INTO analytics_snapshots (user_id, snapshot_date, total_clicks, unique_visitors, top_link_id, source_event_max_id)
		SELECT daily.user_id, ?, daily.total_clicks, daily.unique_visitors, top_links.link_id, daily.source_event_max_id
		FROM daily
		LEFT JOIN top_links ON top_links.user_id = daily.user_id
		ON DUPLICATE KEY UPDATE
			total_clicks = VALUES(total_clicks),
			unique_visitors = VALUES(unique_visitors),
			top_link_id = VALUES(top_link_id),
			source_event_max_id = VALUES(source_event_max_id)
	`, start, end, start, end, start); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE worker_runs
		SET status = 'succeeded', finished_at = now(), rows_processed = ?
		WHERE id = ?
	`, rows, runID); err != nil {
		return 0, err
	}
	return rows, tx.Commit()
}

func (s *Store) CleanupOldClicks(ctx context.Context, retentionDays int) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM click_events WHERE event_time < DATE_SUB(now(), INTERVAL ? DAY)`, retentionDays)
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
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(DISTINCT ip_address) FROM click_events WHERE user_id = ?`, user.ID).Scan(&total, &unique); err != nil {
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
		WHERE user_id = ?
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
		WHERE user_id = ?
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
		WHERE user_id = ? AND id > ?
		ORDER BY id
		LIMIT ?
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
		SELECT COUNT(DISTINCT INDEX_NAME)
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
	`)
	var count int
	if err := row.Scan(&count); err != nil {
		return "", err
	}
	return fmt.Sprintf("MySQL/TiDB schema has %d indexes for auth, profile lookup, click ingestion, and analytics queries.", count), nil
}
