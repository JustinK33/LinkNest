package models

import "time"

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
