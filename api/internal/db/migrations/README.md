# API database migrations

The API store runs on SQLite (default) or PostgreSQL (`--db-driver=postgres`,
built with `-tags postgres`). `db.Store.Migrate` applies migrations at
startup and records each one in `schema_migrations` under its bare filename
(for example `001_init.sql`).

| Directory | Holds | Applied on |
|---|---|---|
| `sqlite/` | Legacy migrations 001–012, SQLite dialect. **Frozen**: never edit these files. | SQLite |
| `postgres/` | Hand-written PostgreSQL equivalents of 001–012, same filenames, same resulting schema. **Frozen.** | PostgreSQL |
| `common/` | **Every new migration.** One portable file that both drivers run unchanged. | Both |

For each driver, `Migrate` takes that driver's legacy set plus `common/`,
sorts them by filename, and applies whatever is not yet recorded. A version
number used in both the legacy set and `common/` is an error.

## Adding a migration

1. Create `common/<NNN>_<name>.sql` with the next free number. Version 012
   (account-removal cleanup) predates this layout and is the last legacy
   per-driver migration, so shared migrations start at 013.
2. Write SQL that both SQLite and PostgreSQL accept. `TestSharedMigrationsArePortable`
   fails the build on these SQLite-only constructs:
   - `datetime(...)`, `strftime(...)`, `julianday(...)`: store timestamps as
     `TEXT` and bind them from Go (RFC 3339 UTC via `time.RFC3339`, or
     `db.NowTimestamp()` for the older `YYYY-MM-DD HH:MM:SS` columns).
   - `AUTOINCREMENT`: portable identity columns differ per engine, so new
     tables should use application-generated `TEXT` keys.
   - `INSERT OR IGNORE` / `INSERT OR REPLACE`: use `INSERT ... ON CONFLICT (...) DO NOTHING / DO UPDATE`.
   - `COLLATE NOCASE`: compare with `LOWER(...)` in the query instead.
   - `?` placeholders: migrations are run as plain DDL/DML with no parameters.
3. Statements are split on `;` followed by a newline. Don't end a comment line
   with `;`, and don't use dollar-quoted function bodies.
4. Migrations are append-only: never edit or renumber one that has shipped.
