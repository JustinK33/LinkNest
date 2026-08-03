package auth

import (
	"testing"
	"time"
)

func TestHashPasswordAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("valid-password-1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "valid-password-1" {
		t.Fatal("password hash must not store plaintext")
	}
	if !CheckPassword(hash, "valid-password-1") {
		t.Fatal("expected password to match")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Fatal("wrong password matched")
	}
}

func TestSignedSessionVerification(t *testing.T) {
	signed := Sign("session-1", "secret")
	value, ok := Verify(signed, "secret")
	if !ok {
		t.Fatal("expected signed value to verify")
	}
	if value != "session-1" {
		t.Fatalf("value = %q", value)
	}
	if _, ok := Verify(signed+"tamper", "secret"); ok {
		t.Fatal("tampered signature verified")
	}
}

func TestSlug(t *testing.T) {
	tests := map[string]string{
		"Justin K":         "justin-k",
		"  Data Engineer ": "data-engineer",
		"Go/Postgres":      "go-postgres",
		"!!!":              "",
	}
	for input, want := range tests {
		if got := Slug(input); got != want {
			t.Fatalf("Slug(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIdempotencyKeyBucketsByMinute(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 30, 12, 0, time.UTC)
	first := IdempotencyKey(10, "127.0.0.1", "ua", "ref", now)
	second := IdempotencyKey(10, "127.0.0.1", "ua", "ref", now.Add(20*time.Second))
	third := IdempotencyKey(10, "127.0.0.1", "ua", "ref", now.Add(70*time.Second))
	if first != second {
		t.Fatal("same minute should produce same idempotency key")
	}
	if first == third {
		t.Fatal("different minute should produce a different idempotency key")
	}
}

func BenchmarkIdempotencyKey(b *testing.B) {
	now := time.Now()
	for i := 0; i < b.N; i++ {
		_ = IdempotencyKey(10, "127.0.0.1", "ua", "ref", now)
	}
}
