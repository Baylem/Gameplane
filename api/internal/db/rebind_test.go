package db

import "testing"

func TestRebind(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"no placeholders", "SELECT 1", "SELECT 1"},
		{"one", "SELECT id FROM users WHERE username = ?", "SELECT id FROM users WHERE username = $1"},
		{"many", "INSERT INTO t(a, b, c) VALUES (?, ?, ?)", "INSERT INTO t(a, b, c) VALUES ($1, $2, $3)"},
		{"double digits",
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)"},
		{"adjacent", "(? = 0 OR id < ?)", "($1 = 0 OR id < $2)"},
		{"string literal kept", "SELECT '?' , ? FROM t", "SELECT '?' , $1 FROM t"},
		{"escaped quote in literal", "SELECT 'it''s ?', ?", "SELECT 'it''s ?', $1"},
		{"backslash literal", `LOWER(actor) LIKE ? ESCAPE '\' AND method = ?`, `LOWER(actor) LIKE $1 ESCAPE '\' AND method = $2`},
		{"quoted identifier", `SELECT "col?" FROM t WHERE x = ?`, `SELECT "col?" FROM t WHERE x = $1`},
		{"line comment", "SELECT ? -- why?\nFROM t WHERE y = ?", "SELECT $1 -- why?\nFROM t WHERE y = $2"},
		{"trailing line comment", "SELECT ? -- what?", "SELECT $1 -- what?"},
		{"block comment", "SELECT /* a ? b */ ? FROM t", "SELECT /* a ? b */ $1 FROM t"},
		{"unterminated literal", "SELECT ?, 'abc?", "SELECT $1, 'abc?"},
		{"unterminated block comment", "SELECT ? /* x ?", "SELECT $1 /* x ?"},
		{"lone dash and slash", "SELECT a - ?, b / ?", "SELECT a - $1, b / $2"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Rebind("postgres", tc.in); got != tc.want {
				t.Errorf("Rebind(postgres, %q) = %q, want %q", tc.in, got, tc.want)
			}
			if got := Rebind("sqlite", tc.in); got != tc.in {
				t.Errorf("Rebind(sqlite, %q) = %q, want it unchanged", tc.in, got)
			}
		})
	}
}

func TestRebind_UnknownDriverIsIdentity(t *testing.T) {
	const q = "SELECT ? FROM t"
	if got := Rebind("", q); got != q {
		t.Fatalf("Rebind(\"\", %q) = %q, want unchanged", q, got)
	}
}
