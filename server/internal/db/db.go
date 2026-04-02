package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

// DB wraps a sql.DB connection with application-specific methods.
type DB struct {
	conn *sql.DB
}

// Open opens a SQLite database at the given path and applies pragmas.
func Open(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Apply pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-20000",
		"PRAGMA temp_store=MEMORY",
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			return nil, fmt.Errorf("exec pragma %q: %w", p, err)
		}
	}

	conn.SetMaxOpenConns(1)

	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Exec runs a raw SQL statement. Used primarily for testing.
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

// MigrationFile represents a single SQL migration file.
type MigrationFile struct {
	Name    string
	Content string
}

// Migrate runs any unapplied migrations in order.
func (db *DB) Migrate(migrations []MigrationFile) error {
	// Ensure schema_migrations exists
	if _, err := db.conn.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, mig := range migrations {
		// Parse version from filename like "001_initial.sql"
		parts := strings.SplitN(mig.Name, "_", 2)
		if len(parts) < 2 {
			continue
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		// Check if already applied
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
		if err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if count > 0 {
			continue
		}

		tx, err := db.conn.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", version, err)
		}

		if _, err := tx.Exec(mig.Content); err != nil {
			tx.Rollback()
			return fmt.Errorf("exec migration %d: %w", version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}

		slog.Info("applied migration", "version", version, "file", mig.Name)
	}

	return nil
}
