package db

import (
	"context"
	"database/sql"
	"errors"
)

// ConfigValue returns the raw JSON value stored for an admin config
// section (the `config` table is a simple key→JSON map written by the
// /admin/config handler) and whether the key was present. The `?`
// placeholder is rebound for Postgres by the driver connection (see Rebind).
func (s *Store) ConfigValue(ctx context.Context, key string) (string, bool, error) {
	return scanConfigValue(s.DB.QueryRowContext(ctx, "SELECT value FROM config WHERE key = ?", key))
}

// ConfigValueTx is ConfigValue but reads through an open transaction, so it
// does not need a second connection (required under sqlite's single-conn cap).
func ConfigValueTx(ctx context.Context, tx *sql.Tx, key string) (string, bool, error) {
	return scanConfigValue(tx.QueryRowContext(ctx, "SELECT value FROM config WHERE key = ?", key))
}

func scanConfigValue(row *sql.Row) (string, bool, error) {
	var v string
	err := row.Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}
