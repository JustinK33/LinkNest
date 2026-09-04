package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"linknest/models"
)

const CookieName = "linknest_session"

const ErrPasswordTooShort = models.UserError("Your password must be at least 8 characters.")

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", ErrPasswordTooShort
	}
	body, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(body), err
}

func CheckPassword(hash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func NewSessionID() (string, error) {
	body := make([]byte, 32)
	if _, err := rand.Read(body); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func Sign(value string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return value + "." + sig
}

func Verify(signed string, secret string) (string, bool) {
	value, sig, ok := strings.Cut(signed, ".")
	if !ok {
		return "", false
	}
	expected := Sign(value, secret)
	_, expectedSig, _ := strings.Cut(expected, ".")
	return value, hmac.Equal([]byte(sig), []byte(expectedSig))
}

func SetSessionCookie(w http.ResponseWriter, sessionID string, secret string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    Sign(sessionID, secret),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// FlashCookieName holds a one-shot "kind|message" pair across a redirect.
// A cookie rather than a query param so a refresh doesn't re-show a stale
// message and it never ends up in a copied URL.
const FlashCookieName = "linknest_flash"

// Flash is what the layout renders. Kind matches the .flash CSS modifiers
// ("notice" or "alert").
type Flash struct {
	Kind    string
	Message string
}

func SetFlash(w http.ResponseWriter, kind string, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:     FlashCookieName,
		Value:    base64.RawURLEncoding.EncodeToString([]byte(kind + "|" + message)),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60,
	})
}

// TakeFlash reads the flash cookie and clears it in the same response, so the
// message is shown exactly once.
func TakeFlash(w http.ResponseWriter, r *http.Request) *Flash {
	cookie, err := r.Cookie(FlashCookieName)
	if err != nil {
		return nil
	}
	http.SetCookie(w, &http.Cookie{
		Name:     FlashCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil
	}
	kind, message, ok := strings.Cut(string(raw), "|")
	if !ok || (kind != "notice" && kind != "alert") {
		return nil
	}
	return &Flash{Kind: kind, Message: message}
}

func Slug(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	var b strings.Builder
	lastDash := false
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func IdempotencyKey(linkID int64, ip string, userAgent string, referrer string, now time.Time) string {
	bucket := now.UTC().Truncate(time.Minute).Format(time.RFC3339)
	raw := fmt.Sprintf("%d|%s|%s|%s|%s", linkID, ip, userAgent, referrer, bucket)
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
