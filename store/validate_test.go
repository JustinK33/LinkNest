package store

import (
	"errors"
	"strings"
	"testing"

	"linknest/models"
)

func TestValidateSignup(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		username    string
		first, last string
		want        error
	}{
		{name: "ok", email: "ada@example.com", username: "ada_lovelace", first: "Ada", last: "Lovelace"},
		{name: "dots and dashes in username", email: "a@b.co", username: "ada.l-1", first: "Ada", last: "L"},

		{name: "no at sign", email: "ada.example.com", username: "ada", first: "Ada", last: "L", want: ErrInvalidEmail},
		{name: "no domain", email: "ada@", username: "ada", first: "Ada", last: "L", want: ErrInvalidEmail},
		{name: "empty email", email: "", username: "ada", first: "Ada", last: "L", want: ErrInvalidEmail},
		{name: "spaces in email", email: "a b@example.com", username: "ada", first: "Ada", last: "L", want: ErrInvalidEmail},
		// mail.ParseAddress accepts a display name, which is not a bare address.
		{name: "display name form", email: "Ada <ada@example.com>", username: "ada", first: "Ada", last: "L", want: ErrInvalidEmail},
		{name: "over length email", email: strings.Repeat("a", 250) + "@example.com", username: "ada", first: "Ada", last: "L", want: ErrInvalidEmail},

		{name: "username too short", email: "a@b.co", username: "ad", first: "Ada", last: "L", want: ErrInvalidUsername},
		{name: "username too long", email: "a@b.co", username: strings.Repeat("a", 31), first: "Ada", last: "L", want: ErrInvalidUsername},
		{name: "username with space", email: "a@b.co", username: "ada l", first: "Ada", last: "L", want: ErrInvalidUsername},
		{name: "username with slash", email: "a@b.co", username: "ada/l", first: "Ada", last: "L", want: ErrInvalidUsername},
		// auth.Slug would turn this into "", which used to be the only guard and
		// would have produced a user with an empty public URL.
		{name: "punctuation only username", email: "a@b.co", username: "...", first: "Ada", last: "L", want: ErrInvalidUsername},

		{name: "missing first name", email: "a@b.co", username: "ada", first: "", last: "L", want: ErrNameRequired},
		{name: "missing last name", email: "a@b.co", username: "ada", first: "Ada", last: "", want: ErrNameRequired},
		{name: "name too long", email: "a@b.co", username: "ada", first: strings.Repeat("a", 61), last: "L", want: ErrNameTooLong},
		// Counted in runes, not bytes: 60 three-byte characters is within the limit.
		{name: "multibyte name at limit", email: "a@b.co", username: "ada", first: strings.Repeat("あ", 60), last: "L"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateSignup(tc.email, tc.username, tc.first, tc.last)
			if !errors.Is(got, tc.want) {
				t.Fatalf("validateSignup(%q, %q, %q, %q) = %v, want %v", tc.email, tc.username, tc.first, tc.last, got, tc.want)
			}
		})
	}
}

func TestValidateLink(t *testing.T) {
	tests := []struct {
		name  string
		title string
		url   string
		want  error
	}{
		{name: "https", title: "Portfolio", url: "https://example.com/work"},
		{name: "http", title: "Portfolio", url: "http://example.com"},
		{name: "mailto", title: "Email", url: "mailto:ada@example.com"},
		{name: "tel", title: "Phone", url: "tel:+15550100"},
		{name: "root relative", title: "Resume", url: "/resume.pdf"},

		{name: "no title", title: "", url: "https://example.com", want: ErrTitleRequired},
		{name: "title too long", title: strings.Repeat("a", 121), url: "https://example.com", want: ErrTitleTooLong},

		{name: "no url", title: "T", url: "", want: ErrInvalidLinkURL},
		{name: "scheme with no host", title: "T", url: "https://", want: ErrInvalidLinkURL},
		{name: "unsupported scheme", title: "T", url: "javascript:alert(1)", want: ErrInvalidLinkURL},
		{name: "empty mailto", title: "T", url: "mailto:", want: ErrInvalidLinkURL},
		// Looks same-site but leaves the site, so it is not the relative form
		// safeURL deliberately allows through.
		{name: "protocol relative", title: "T", url: "//evil.example", want: ErrInvalidLinkURL},
		{name: "url too long", title: "T", url: "https://example.com/" + strings.Repeat("a", 2048), want: ErrInvalidLinkURL},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := validateLink(tc.title, tc.url); !errors.Is(got, tc.want) {
				t.Fatalf("validateLink(%q, %q) = %v, want %v", tc.title, tc.url, got, tc.want)
			}
		})
	}
}

// Every validation error must be renderable, otherwise the handler logs it and
// shows a generic message instead of telling the user what to fix.
func TestValidationErrorsAreUserFacing(t *testing.T) {
	errs := []error{
		ErrEmailTaken, ErrUsernameTaken, ErrInvalidEmail, ErrInvalidUsername,
		ErrNameRequired, ErrNameTooLong, ErrBioTooLong, ErrTitleRequired,
		ErrTitleTooLong, ErrInvalidLinkURL, ErrLinkNotFound, ErrLinkAtEdge,
	}
	for _, err := range errs {
		var friendly models.UserError
		if !errors.As(err, &friendly) {
			t.Errorf("%v is not a models.UserError, so it would be hidden behind a generic message", err)
			continue
		}
		msg := err.Error()
		if !strings.HasSuffix(msg, ".") || strings.ToUpper(msg[:1]) != msg[:1] {
			t.Errorf("%q should read as a sentence: capitalised and ending in a period", msg)
		}
	}
}
