package app

import (
	"html/template"
	"strings"
	"testing"

	"linknest/auth"
	"linknest/models"
	"linknest/web"
)

// pageData supplies the minimum each page dereferences, so a renamed or
// misspelled field fails here instead of at request time.
func pageData(name string) any {
	user := models.User{ID: 1, Slug: "ada", FirstName: "Ada", LastName: "Lovelace", ProfileColor: "#56738c"}
	switch name {
	case "profile.html":
		return map[string]any{
			"User":  user,
			"Links": []models.Link{{ID: 1, Title: "Portfolio", URL: "https://www.example.com/work"}},
		}
	case "dashboard.html":
		return models.Dashboard{User: user}
	case "error.html":
		return map[string]any{"Status": 404, "Title": "Not Found", "Message": "nope"}
	default:
		return map[string]any{}
	}
}

func TestEveryPageRenders(t *testing.T) {
	for _, name := range PageNames {
		tmpl := template.Must(template.New(name).Funcs(Funcs).ParseFS(web.Templates, "templates/layout.html", "templates/"+name))
		page := struct {
			CurrentUser models.User
			Flash       *auth.Flash
			SiteURL     string
			Data        any
		}{Flash: &auth.Flash{Kind: "notice", Message: "Saved."}, SiteURL: "https://linknest.test", Data: pageData(name)}

		var out strings.Builder
		if err := tmpl.ExecuteTemplate(&out, "layout", page); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !strings.Contains(out.String(), `class="flash notice"`) {
			t.Errorf("%s: layout did not render the flash", name)
		}
		if strings.Contains(out.String(), "<no value>") {
			t.Errorf("%s: rendered a missing field as <no value>", name)
		}
	}
}

// The fonts must be reachable from the document, not from an @import placed
// after a ruleset in app.css, where every browser silently drops it.
func TestFontsAreLoadedFromTheLayout(t *testing.T) {
	layout, err := web.Templates.ReadFile("templates/layout.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(layout), "fonts.googleapis.com/css2") {
		t.Error("layout.html does not link the webfont stylesheet")
	}

	css, err := web.Static.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(css), "@import") {
		t.Error("app.css has an @import again; it is dropped when it follows a ruleset")
	}
}

// The dark scheme works by overriding tokens, so a rule that hardcodes an opaque
// colour is invisible in one scheme or the other. Translucent rgba() is fine: it
// composites over whatever is underneath and adapts on its own.
func TestNoOpaqueColoursOutsideTheTokenBlocks(t *testing.T) {
	css, err := web.Static.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	body := outsideTokenBlocks(t, string(css))

	for i, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "*") || strings.Contains(line, "/*") {
			continue // a hex in prose is describing the problem, not causing it
		}
		if strings.Contains(line, "#") && strings.ContainsAny(line, "0123456789abcdefABCDEF") {
			if idx := strings.Index(line, "#"); idx >= 0 && isHexColour(line[idx:]) {
				t.Errorf("line %d hardcodes %s; define a token in :root and override it in the dark block instead", i+1, strings.TrimSpace(line))
			}
		}
		// "color: white" is legitimate on the bands pinned dark in both schemes.
		// A white *fill* is not: it is a light surface by another name.
		if strings.Contains(line, "background") && (strings.Contains(line, " white") || strings.Contains(line, " black")) {
			t.Errorf("line %d fills with a fixed %s; use --bg-card or --surface-inverted", i+1, strings.TrimSpace(line))
		}
	}
}

// outsideTokenBlocks returns app.css with :root and the prefers-color-scheme
// block removed, which are the two places a literal colour belongs.
func outsideTokenBlocks(t *testing.T, css string) string {
	t.Helper()
	cut := func(start string, close string) {
		i := strings.Index(css, start)
		if i < 0 {
			t.Fatalf("app.css no longer contains %q", start)
		}
		j := strings.Index(css[i:], close)
		if j < 0 {
			t.Fatalf("%q is not closed by %q", start, close)
		}
		css = css[:i] + css[i+j+len(close):]
	}
	cut("@media (prefers-color-scheme: dark)", "\n  }\n}")
	cut(":root {", "\n}")
	return css
}

// isHexColour reports whether s starts with #rgb, #rgba, #rrggbb or #rrggbbaa.
func isHexColour(s string) bool {
	digits := 0
	for _, r := range s[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			break
		}
		digits++
	}
	return digits == 3 || digits == 4 || digits == 6 || digits == 8
}

// Every var(--token) in app.css must resolve, or the whole declaration is
// dropped. A missing --space-14 is what silently collapsed .landing-nfc-band.
func TestEveryCSSVariableIsDefined(t *testing.T) {
	css, err := web.Static.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	text := string(css)

	defined := map[string]bool{}
	for _, part := range strings.Split(text, "--")[1:] {
		if name, rest, ok := strings.Cut(part, ":"); ok && !strings.ContainsAny(name, " \t\n(){};") {
			// A definition is "--name:", a usage is "var(--name)".
			if !strings.HasPrefix(strings.TrimSpace(rest), ")") {
				defined["--"+name] = true
			}
		}
	}

	for _, part := range strings.Split(text, "var(")[1:] {
		name, _, ok := strings.Cut(part, ")")
		if !ok {
			continue
		}
		name, _, _ = strings.Cut(name, ",") // var(--x, fallback)
		name = strings.TrimSpace(name)
		if !defined[name] {
			t.Errorf("app.css uses %s but never defines it", name)
		}
	}
}
