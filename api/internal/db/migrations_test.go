package db

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// sqliteOnlyConstructs are the SQLite-dialect constructs a shared
// (migrations/common) migration must not use, because PostgreSQL rejects
// them or gives them a different meaning. See migrations/README.md.
var sqliteOnlyConstructs = []struct {
	name string
	re   *regexp.Regexp
}{
	{"datetime()", regexp.MustCompile(`(?i)\bdatetime\s*\(`)},
	{"strftime()", regexp.MustCompile(`(?i)\bstrftime\s*\(`)},
	{"julianday()", regexp.MustCompile(`(?i)\bjulianday\s*\(`)},
	{"AUTOINCREMENT", regexp.MustCompile(`(?i)\bautoincrement\b`)},
	{"INSERT OR ...", regexp.MustCompile(`(?i)\binsert\s+or\s+\w+`)},
	{"COLLATE NOCASE", regexp.MustCompile(`(?i)\bcollate\s+nocase\b`)},
}

// portabilityProblems returns every reason sqlText can't be a shared
// migration. Comments are ignored for the construct checks (they may name
// the constructs they avoid) but a comment line ending in ';' is itself a
// problem, since the migrator splits statements on ";\n".
func portabilityProblems(sqlText string) []string {
	var problems []string
	var code strings.Builder
	for line := range strings.SplitSeq(sqlText, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			if strings.HasSuffix(trimmed, ";") {
				problems = append(problems, "comment line ends with ';': "+trimmed)
			}
			continue
		}
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		code.WriteString(line)
		code.WriteByte('\n')
	}
	body := code.String()
	for _, c := range sqliteOnlyConstructs {
		if c.re.MatchString(body) {
			problems = append(problems, "uses SQLite-only "+c.name)
		}
	}
	if Rebind("postgres", body) != body {
		problems = append(problems, "uses a ? placeholder")
	}
	return problems
}

func TestPortabilityProblems_DetectsSQLiteOnlyConstructs(t *testing.T) {
	bad := map[string]string{
		"datetime":       "CREATE TABLE t (ts TEXT NOT NULL DEFAULT (datetime('now')));\n",
		"strftime":       "UPDATE t SET ts = strftime('%Y', 'now');\n",
		"julianday":      "SELECT julianday('now');\n",
		"autoincrement":  "CREATE TABLE t (id INTEGER PRIMARY KEY AUTOINCREMENT);\n",
		"insert or":      "INSERT OR IGNORE INTO t(a) VALUES (1);\n",
		"insert or repl": "insert or replace into t(a) values (1);\n",
		"nocase":         "CREATE TABLE t (name TEXT COLLATE NOCASE);\n",
		"placeholder":    "INSERT INTO t(a) VALUES (?);\n",
		"comment semi":   "-- first do this;\nCREATE TABLE t (a TEXT);\n",
	}
	for name, sqlText := range bad {
		if len(portabilityProblems(sqlText)) == 0 {
			t.Errorf("%s: expected a portability problem in %q", name, sqlText)
		}
	}

	good := "-- Avoids datetime('now') and AUTOINCREMENT on purpose.\n" +
		"CREATE TABLE t (\n    id TEXT PRIMARY KEY, -- app-generated, not AUTOINCREMENT\n    ts TEXT NOT NULL\n);\n" +
		"INSERT INTO t(id, ts) VALUES ('a', '2026-01-01T00:00:00Z') ON CONFLICT (id) DO NOTHING;\n"
	if p := portabilityProblems(good); len(p) != 0 {
		t.Errorf("portable SQL flagged: %v", p)
	}
}

// TestSharedMigrationsArePortable fails on any migrations/common file that
// uses a SQLite-only construct: those files run unchanged on both drivers.
func TestSharedMigrationsArePortable(t *testing.T) {
	files, err := listSQL(migrations, sharedMigrationDir)
	if err != nil {
		t.Fatalf("list shared migrations: %v", err)
	}
	for _, f := range files {
		content, err := fs.ReadFile(migrations, f.path)
		if err != nil {
			t.Fatalf("read %s: %v", f.path, err)
		}
		for _, p := range portabilityProblems(string(content)) {
			t.Errorf("%s: %s", f.path, p)
		}
	}
}

// TestLegacyMigrationSetsMatch: the SQLite and Postgres legacy sets must
// hold the same versions, so both drivers record the same schema_migrations
// rows.
func TestLegacyMigrationSetsMatch(t *testing.T) {
	sqliteFiles, err := listSQL(migrations, legacyMigrationDirs["sqlite"])
	if err != nil {
		t.Fatalf("list sqlite: %v", err)
	}
	pgFiles, err := listSQL(migrations, legacyMigrationDirs["postgres"])
	if err != nil {
		t.Fatalf("list postgres: %v", err)
	}
	names := func(files []migrationFile) string {
		out := make([]string, len(files))
		for i, f := range files {
			out[i] = f.name
		}
		return strings.Join(out, ",")
	}
	if names(sqliteFiles) != names(pgFiles) {
		t.Fatalf("legacy sets differ:\n sqlite:   %s\n postgres: %s", names(sqliteFiles), names(pgFiles))
	}
	if len(sqliteFiles) == 0 {
		t.Fatal("no legacy migrations embedded")
	}
}

// TestSharedMigrationsFollowLegacy: shared migrations are numbered after
// every legacy one, so "legacy set, then shared set" is also version order.
func TestSharedMigrationsFollowLegacy(t *testing.T) {
	var maxLegacy string
	for _, dir := range legacyMigrationDirs {
		files, err := listSQL(migrations, dir)
		if err != nil {
			t.Fatalf("list %s: %v", dir, err)
		}
		for _, f := range files {
			if v := migrationVersion(f.name); v > maxLegacy {
				maxLegacy = v
			}
		}
	}
	shared, err := listSQL(migrations, sharedMigrationDir)
	if err != nil {
		t.Fatalf("list shared: %v", err)
	}
	for _, f := range shared {
		v := migrationVersion(f.name)
		if len(v) != 3 || v == f.name {
			t.Errorf("%s: name must start with a three-digit version", f.path)
		}
		if v <= maxLegacy || v < "013" {
			t.Errorf("%s: shared migrations must be numbered 013 or later and after legacy %s", f.path, maxLegacy)
		}
	}
}

func TestMigrationFiles_OrderAndCollision(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/sqlite/001_a.sql":   {Data: []byte("SELECT 1;\n")},
		"migrations/sqlite/002_b.sql":   {Data: []byte("SELECT 1;\n")},
		"migrations/postgres/001_a.sql": {Data: []byte("SELECT 1;\n")},
		"migrations/postgres/002_b.sql": {Data: []byte("SELECT 1;\n")},
		"migrations/common/004_d.sql":   {Data: []byte("SELECT 1;\n")},
		"migrations/common/003_c.sql":   {Data: []byte("SELECT 1;\n")},
		"migrations/common/README.md":   {Data: []byte("docs")},
	}
	for _, driver := range []string{"sqlite", "postgres"} {
		files, err := migrationFiles(fsys, driver)
		if err != nil {
			t.Fatalf("%s: %v", driver, err)
		}
		var got []string
		for _, f := range files {
			got = append(got, f.path)
		}
		want := []string{
			"migrations/" + driver + "/001_a.sql",
			"migrations/" + driver + "/002_b.sql",
			"migrations/common/003_c.sql",
			"migrations/common/004_d.sql",
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s: order = %v, want %v", driver, got, want)
		}
	}

	fsys["migrations/common/002_clash.sql"] = &fstest.MapFile{Data: []byte("SELECT 1;\n")}
	if _, err := migrationFiles(fsys, "sqlite"); err == nil || !strings.Contains(err.Error(), "version 002") {
		t.Fatalf("expected a version collision error, got %v", err)
	}
}

func TestMigrationFiles_UnknownDriver(t *testing.T) {
	if _, err := migrationFiles(migrations, "mysql"); err == nil {
		t.Fatal("expected an error for a driver with no migrations")
	}
}

func TestMigrationFiles_MissingSharedDirIsEmpty(t *testing.T) {
	fsys := fstest.MapFS{"migrations/sqlite/001_a.sql": {Data: []byte("SELECT 1;\n")}}
	files, err := migrationFiles(fsys, "sqlite")
	if err != nil || len(files) != 1 {
		t.Fatalf("files=%v err=%v", files, err)
	}
}

func TestMigrationVersion(t *testing.T) {
	for in, want := range map[string]string{
		"001_init.sql":  "001",
		"013_x.sql":     "013",
		"noversion.sql": "noversion.sql",
	} {
		if got := migrationVersion(in); got != want {
			t.Errorf("migrationVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsCommentOnly(t *testing.T) {
	for in, want := range map[string]bool{
		"":                             true,
		"   \n\t":                      true,
		"-- a\n  -- b\n":               true,
		"-- a\nSELECT 1":               false,
		"CREATE TABLE t (a TEXT) -- x": false,
	} {
		if got := isCommentOnly(in); got != want {
			t.Errorf("isCommentOnly(%q) = %v, want %v", in, got, want)
		}
	}
}

// TestMigrate_SQLiteRecordsLegacyVersions pins what an SQLite database
// records: exactly the pre-split filenames, so existing installs see no new
// or re-run migrations after the move into migrations/sqlite, and applied_at
// keeps datetime('now')'s "YYYY-MM-DD HH:MM:SS" form.
func TestMigrate_SQLiteRecordsLegacyVersions(t *testing.T) {
	s := newRBACStore(t)
	rows, err := s.DB.QueryContext(t.Context(), `SELECT version, applied_at FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var got []string
	for rows.Next() {
		var v, at string
		if err := rows.Scan(&v, &at); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if _, err := time.Parse(sqliteTimestampLayout, at); err != nil {
			t.Errorf("%s applied_at %q is not %q: %v", v, at, sqliteTimestampLayout, err)
		}
		got = append(got, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	want := []string{
		"001_init.sql", "002_config.sql", "003_roles.sql", "004_cluster_rbac.sql",
		"005_audit_chain.sql", "006_share_links.sql", "007_audit_reason.sql",
		"008_captures_rbac.sql", "009_share_links_cluster.sql",
		"010_share_links_expiry_nullable.sql", "011_user_theme_preferences.sql",
	}
	shared, err := listSQL(migrations, sharedMigrationDir)
	if err != nil {
		t.Fatalf("list shared: %v", err)
	}
	for _, f := range shared {
		want = append(want, f.name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("recorded versions = %v, want %v", got, want)
	}
}

func TestNowTimestamp_Format(t *testing.T) {
	ts := NowTimestamp()
	parsed, err := time.Parse(sqliteTimestampLayout, ts)
	if err != nil {
		t.Fatalf("NowTimestamp() = %q: %v", ts, err)
	}
	if d := time.Since(parsed); d < -time.Minute || d > time.Minute {
		t.Fatalf("NowTimestamp() = %q is not the current UTC time", ts)
	}
}
