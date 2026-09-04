// Command preview renders the real templates with fake data so the UI can be
// inspected without a database. Throwaway dev tool, not part of the app.
package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"linknest/app"
	"linknest/auth"
	"linknest/models"
	"linknest/web"
)

func main() {
	user := models.User{ID: 1, Slug: "ada", Username: "ada", FirstName: "Ada", LastName: "Lovelace",
		Bio: "Mathematician. Writing about analytical engines, computation and the occasional poem.", ProfileColor: "#56738c"}
	links := []models.Link{
		{ID: 1, Title: "Portfolio", URL: "https://www.example.com/work", Public: true, ClickCount: 1284},
		{ID: 2, Title: "Notes on the Analytical Engine", URL: "https://notes.example.org/engine", Public: true, ClickCount: 431},
		{ID: 3, Title: "Draft (private)", URL: "https://example.com/draft", Public: false, ClickCount: 3},
	}
	dashboard := models.Dashboard{
		User: user, Links: links, TotalClicks: 1718, UniqueVisitors: 902,
		TopLinks: links[:2],
		Daily: []models.DailyStat{
			{Date: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), TotalClicks: 210, UniqueVisitors: 120, TopLinkID: 1, TopLinkClicks: 140},
			{Date: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), TotalClicks: 265, UniqueVisitors: 143, TopLinkID: 1, TopLinkClicks: 171},
			{Date: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), TotalClicks: 301, UniqueVisitors: 166, TopLinkID: 2, TopLinkClicks: 188},
		},
	}

	data := map[string]any{
		"home.html":      nil,
		"login.html":     map[string]any{},
		"register.html":  map[string]any{"Error": "That email is already registered. Try signing in instead.", "Username": "ada", "FirstName": "Ada", "LastName": "Lovelace", "Email": "ada@example.com"},
		"dashboard.html": dashboard,
		"profile.html":   map[string]any{"User": user, "Links": links[:2]},
		"error.html":     map[string]any{"Status": 404, "Title": "Not Found", "Message": "We couldn't find that page. The link may have changed or the profile may no longer exist."},
	}
	// Signed-out chrome for the pages a visitor actually sees that way.
	signedOut := map[string]bool{"login.html": true, "register.html": true, "profile.html": true}
	flash := map[string]*auth.Flash{"dashboard.html": {Kind: "notice", Message: `Added "Portfolio" to your profile.`}}

	pages := map[string]*template.Template{}
	for _, name := range app.PageNames {
		pages[name] = template.Must(template.New(name).Funcs(app.Funcs).ParseFS(web.Templates, "templates/layout.html", "templates/"+name))
	}

	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))
	// /frame?w=375&p=home renders a page in a fixed-width iframe. Headless Chrome
	// clamps its own window to 500px on macOS, so this is the only way to lay a
	// page out at a real phone width. Same origin, so the probe result can be
	// copied out to the top-level DOM where --dump-dom can see it.
	http.HandleFunc("GET /frame", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// frag reaches the child as a real location.hash, which is how the
		// dashboard decides which tab to open.
		frag := ""
		if f := q.Get("frag"); f != "" {
			frag = "#" + f
		}
		fmt.Fprintf(w, framePage, q.Get("w"), q.Get("p"), q.Encode(), frag)
	})
	http.HandleFunc("GET /{page}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("page") + ".html"
		tmpl, ok := pages[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		current := user
		if signedOut[name] || r.URL.Query().Has("anon") {
			current = models.User{}
		}
		page := struct {
			CurrentUser models.User
			Flash       *auth.Flash
			SiteURL     string
			Data        any
		}{CurrentUser: current, Flash: flash[name], SiteURL: "http://localhost:8099", Data: data[name]}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "layout", page); err != nil {
			log.Printf("%s: %v", name, err)
			return
		}
		// ?probe reports which elements are wider than the viewport, so a
		// horizontal-overflow bug can be attributed instead of eyeballed.
		if r.URL.Query().Has("probe") {
			_, _ = w.Write([]byte(overflowProbe))
		}
		// ?scroll=N pulls the page up N pixels. Headless Chrome screenshots the
		// window, not the document, so this is how a long page gets captured in
		// legible sections instead of one downscaled strip.
		if n := r.URL.Query().Get("scroll"); n != "" {
			fmt.Fprintf(w, `<style>body{margin-top:-%spx}</style>`, n)
		}
		// ?open expands every <details> so collapsed UI is visible in a screenshot.
		if r.URL.Query().Has("open") {
			_, _ = w.Write([]byte(`<script>document.querySelectorAll("details").forEach(function (d) { d.open = true; });</script>`))
		}
	})
	log.Println("preview on :8099")
	log.Fatal(http.ListenAndServe(":8099", nil))
}

const framePage = `<!doctype html><meta charset="utf-8"><style>
body{margin:0;background:#333}
iframe{width:%spx;height:6000px;border:0;background:#fff;display:block}
pre{color:#0f0;font:12px monospace;padding:8px;margin:0}
</style>
<iframe id="f" src="/%s?%s%s"></iframe>
<script>
document.getElementById('f').addEventListener('load', function () {
  var self = this;
  setTimeout(function () {
    var probe = self.contentDocument.getElementById('overflow-probe');
    var pre = document.createElement('pre');
    pre.id = 'frame-probe';
    pre.textContent = probe ? probe.textContent : '(no probe)';
    document.body.appendChild(pre);
  }, 500);
});
</script>`

const overflowProbe = `<script>
window.addEventListener('load', function () {
  var vw = document.documentElement.clientWidth;
  var out = ['viewport=' + vw + ' scrollWidth=' + document.documentElement.scrollWidth];
  document.querySelectorAll('*').forEach(function (el) {
    var r = el.getBoundingClientRect();
    if (r.width > vw + 1 || r.right > vw + 1) {
      out.push(el.tagName + '.' + (el.className || '(none)') +
        ' w=' + Math.round(r.width) + ' left=' + Math.round(r.left) + ' right=' + Math.round(r.right));
    }
  });
  var pre = document.createElement('pre');
  pre.id = 'overflow-probe';
  pre.textContent = out.join('\n');
  document.body.appendChild(pre);
});
</script>`
