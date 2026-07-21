package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const CookieName = "linknest_session"

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
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
