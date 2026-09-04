package models

import "time"

// UserError is an error whose text is already written for a person and is safe
// to render in a page. Anything that is not a UserError gets logged and shown as
// a generic failure, which is what keeps driver text like
// "Error 1062: Duplicate entry ... for key 'idx_users_email'" out of the UI.
//
// It lives here because both auth and store return these and models is the only
// package both can import.
type UserError string

func (e UserError) Error() string { return string(e) }

type User struct {
	ID           int64
	Email        string
	Username     string
	Slug         string
	FirstName    string
	LastName     string
	Bio          string
	ProfileColor string
}

type Link struct {
	ID         int64
	UserID     int64
	Title      string
	URL        string
	Position   int
	Public     bool
	IconColor  string
	ClickCount int64
}

type ClickEvent struct {
	ID             int64
	IdempotencyKey string
	UserID         int64
	LinkID         int64
	Referrer       string
	UserAgent      string
	IPAddress      string
	CountryCode    string
	DeviceType     string
	BrowserName    string
	EventTime      time.Time
}

type Dashboard struct {
	User           User
	Links          []Link
	TotalClicks    int64
	UniqueVisitors int64
	TopLinks       []Link
	Daily          []DailyStat
}

type DailyStat struct {
	Date           time.Time
	TotalClicks    int64
	UniqueVisitors int64
	TopLinkID      int64
	TopLinkClicks  int64
}
