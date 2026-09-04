package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"linknest/config"
	"linknest/db"
)

// The reorder, update and delete SQL is the only logic in this package that
// cannot be checked without a server: it depends on transaction behaviour and on
// UPDATE ... WHERE user_id matching zero rows. Set LINKNEST_TEST_DATABASE_URL to
// a scratch database to run it, for example
//
//	docker run -d --name linknest-test-mysql -e MYSQL_ALLOW_EMPTY_PASSWORD=yes \
//	  -e MYSQL_DATABASE=linknest_test -p 13306:3306 mysql:8
//	LINKNEST_TEST_DATABASE_URL=mysql://root@127.0.0.1:13306/linknest_test go test ./store/
//
// It truncates every table it touches, so never point it at anything real.
func testStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("LINKNEST_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set LINKNEST_TEST_DATABASE_URL to run the database tests")
	}

	pool, err := db.Open(config.Config{DatabaseURL: url, MaxOpenConns: 4, MaxIdleConns: 2, ConnMaxLifetime: time.Minute})
	if err != nil {
		t.Fatalf("open %s: %v", url, err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Read the table list rather than hardcoding it, so adding a migration does
	// not leave rows behind and make a later test fail for no visible reason.
	rows, err := pool.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()")
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		t.Fatal(err)
	}
	for _, table := range tables {
		if _, err := pool.ExecContext(ctx, "DELETE FROM `"+table+"`"); err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}
	if _, err := pool.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		t.Fatal(err)
	}
	return New(pool)
}

// titles is the list as a visitor would see it, which is the only ordering that
// matters: LinksForUser sorts by position.
func titles(t *testing.T, s *Store, userID int64) []string {
	t.Helper()
	links, err := s.LinksForUser(context.Background(), userID, false, 100, 0)
	if err != nil {
		t.Fatalf("LinksForUser: %v", err)
	}
	out := make([]string, len(links))
	for i, link := range links {
		out[i] = link.Title
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func seed(t *testing.T, s *Store, email, username string, linkTitles ...string) int64 {
	t.Helper()
	ctx := context.Background()
	user, err := s.CreateUser(ctx, email, "correct horse battery", username, "Ada", "Lovelace")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	for _, title := range linkTitles {
		if err := s.CreateLink(ctx, user.ID, title, "https://example.com/"+title, true); err != nil {
			t.Fatalf("CreateLink %q: %v", title, err)
		}
	}
	return user.ID
}

func TestMoveLink(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	userID := seed(t, s, "ada@example.com", "ada", "a", "b", "c")

	if got := titles(t, s, userID); !equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("after create = %v, want [a b c]", got)
	}

	links, _ := s.LinksForUser(ctx, userID, false, 100, 0)
	a, b, c := links[0].ID, links[1].ID, links[2].ID

	if err := s.MoveLink(ctx, userID, c, true); err != nil {
		t.Fatalf("move c up: %v", err)
	}
	if got := titles(t, s, userID); !equal(got, []string{"a", "c", "b"}) {
		t.Fatalf("after c up = %v, want [a c b]", got)
	}

	if err := s.MoveLink(ctx, userID, a, false); err != nil {
		t.Fatalf("move a down: %v", err)
	}
	if got := titles(t, s, userID); !equal(got, []string{"c", "a", "b"}) {
		t.Fatalf("after a down = %v, want [c a b]", got)
	}

	// Both edges must refuse rather than silently do nothing, otherwise the page
	// comes back unchanged with a success message on it.
	if err := s.MoveLink(ctx, userID, c, true); !errors.Is(err, ErrLinkAtEdge) {
		t.Errorf("move first up = %v, want ErrLinkAtEdge", err)
	}
	if err := s.MoveLink(ctx, userID, b, false); !errors.Is(err, ErrLinkAtEdge) {
		t.Errorf("move last down = %v, want ErrLinkAtEdge", err)
	}
	if got := titles(t, s, userID); !equal(got, []string{"c", "a", "b"}) {
		t.Errorf("a refused move changed the order to %v", got)
	}
}

// Rows written before positions existed all share position 0. A pairwise swap
// between two equal positions is a no-op, which is why MoveLink renumbers.
func TestMoveLinkRepairsTiedPositions(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	userID := seed(t, s, "ada@example.com", "ada", "a", "b", "c")

	if _, err := s.DB().ExecContext(ctx, "UPDATE links SET position = 0 WHERE user_id = ?", userID); err != nil {
		t.Fatalf("tie positions: %v", err)
	}
	links, _ := s.LinksForUser(ctx, userID, false, 100, 0)
	last := links[2].ID

	if err := s.MoveLink(ctx, userID, last, true); err != nil {
		t.Fatalf("move up: %v", err)
	}
	if got := titles(t, s, userID); !equal(got, []string{"a", "c", "b"}) {
		t.Fatalf("after move = %v, want [a c b]", got)
	}

	var distinct int
	if err := s.DB().QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT position) FROM links WHERE user_id = ?", userID).Scan(&distinct); err != nil {
		t.Fatal(err)
	}
	if distinct != 3 {
		t.Errorf("positions still collide: %d distinct values for 3 links", distinct)
	}
}

func TestUpdateAndDeleteLinkAreScopedToTheOwner(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	owner := seed(t, s, "ada@example.com", "ada", "a", "b")
	intruder := seed(t, s, "eve@example.com", "eve")

	links, _ := s.LinksForUser(ctx, owner, false, 100, 0)
	target := links[0].ID

	if err := s.UpdateLink(ctx, intruder, target, "hijacked", "https://evil.example", true); !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("cross-account update = %v, want ErrLinkNotFound", err)
	}
	if err := s.DeleteLink(ctx, intruder, target); !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("cross-account delete = %v, want ErrLinkNotFound", err)
	}
	if err := s.MoveLink(ctx, intruder, target, false); !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("cross-account move = %v, want ErrLinkNotFound", err)
	}
	if got := titles(t, s, owner); !equal(got, []string{"a", "b"}) {
		t.Fatalf("owner's links were touched: %v", got)
	}

	if err := s.UpdateLink(ctx, owner, target, "renamed", "https://example.com/renamed", false); err != nil {
		t.Fatalf("UpdateLink: %v", err)
	}
	links, _ = s.LinksForUser(ctx, owner, false, 100, 0)
	if links[0].Title != "renamed" || links[0].Public {
		t.Errorf("update did not apply: %+v", links[0])
	}
	// Public is a checkbox, so an unchecked box arrives as an absent field. The
	// link has to disappear from the public list, not just from the dashboard.
	if got := titles(t, s, owner); len(got) != 2 {
		t.Errorf("dashboard list = %v, want both links", got)
	}
	public, err := s.LinksForUser(ctx, owner, true, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(public) != 1 || public[0].Title != "b" {
		t.Errorf("public list = %+v, want only b", public)
	}

	if err := s.DeleteLink(ctx, owner, target); err != nil {
		t.Fatalf("DeleteLink: %v", err)
	}
	if got := titles(t, s, owner); !equal(got, []string{"b"}) {
		t.Errorf("after delete = %v, want [b]", got)
	}
	if err := s.DeleteLink(ctx, owner, target); !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("second delete = %v, want ErrLinkNotFound", err)
	}
}

// Validation lives in the store so no write path can skip it, including one that
// reaches UpdateLink without going through the handler.
func TestUpdateLinkValidates(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	userID := seed(t, s, "ada@example.com", "ada", "a")
	links, _ := s.LinksForUser(ctx, userID, false, 100, 0)

	if err := s.UpdateLink(ctx, userID, links[0].ID, "a", "javascript:alert(1)", true); !errors.Is(err, ErrInvalidLinkURL) {
		t.Errorf("javascript: url = %v, want ErrInvalidLinkURL", err)
	}
	if err := s.UpdateLink(ctx, userID, links[0].ID, "   ", "https://example.com", true); !errors.Is(err, ErrTitleRequired) {
		t.Errorf("blank title = %v, want ErrTitleRequired", err)
	}
}
