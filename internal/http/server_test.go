package http

import "testing"

func TestDeviceType(t *testing.T) {
	tests := map[string]string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)": "mobile",
		"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X)":          "tablet",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X)":                "desktop",
	}
	for ua, want := range tests {
		if got := deviceType(ua); got != want {
			t.Fatalf("deviceType(%q) = %q, want %q", ua, got, want)
		}
	}
}

func TestBrowserNamePrefersEdgeBeforeChrome(t *testing.T) {
	ua := "Mozilla/5.0 AppleWebKit Chrome/120 Safari/537.36 Edg/120"
	if got := browserName(ua); got != "Edge" {
		t.Fatalf("browserName = %q, want Edge", got)
	}
}

func TestSafeURL(t *testing.T) {
	tests := map[string]string{
		"example.com":          "https://example.com",
		"https://example.com":  "https://example.com",
		"mailto:a@example.com": "mailto:a@example.com",
		"/resume.pdf":          "/resume.pdf",
	}
	for input, want := range tests {
		if got := safeURL(input); got != want {
			t.Fatalf("safeURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestReservedSlug(t *testing.T) {
	if !reservedSlug("api") {
		t.Fatal("api should be reserved")
	}
	if reservedSlug("justin") {
		t.Fatal("justin should not be reserved")
	}
}
