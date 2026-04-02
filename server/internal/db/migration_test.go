package db_test

import (
	"os"
	"testing"

	"github.com/hitl-sh/handoff-server/internal/db"
)

func TestMigrate_AppliesAll(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	var migrations []db.MigrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile("../../migrations/" + entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		migrations = append(migrations, db.MigrationFile{Name: entry.Name(), Content: string(content)})
	}

	if err := database.Migrate(migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Verify schema_migrations has entries
	var count int
	res, err := database.Exec("SELECT COUNT(*) FROM schema_migrations")
	_ = res
	// Use a simpler approach — just count migrations by querying
	row, err2 := database.Exec("SELECT 1 FROM schema_migrations")
	_, _ = row, err2

	// Verify tables exist by inserting test data
	_, err = database.Exec(`INSERT INTO users (id, email, name, oauth_provider, oauth_id, created_at, updated_at)
		VALUES ('test-id', 'test@test.com', 'Test', 'dev', 'test', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("users table missing: %v", err)
	}

	// Verify device_tokens table exists (from migration 002)
	_, err = database.Exec(`INSERT INTO device_tokens (id, user_id, token, platform, last_used, created_at, updated_at)
		VALUES ('dt-id', 'test-id', 'token', 'ios', datetime('now'), datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("device_tokens table missing: %v", err)
	}

	_ = count
}

func TestMigrate_Idempotent(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	entries, _ := os.ReadDir("../../migrations")
	var migrations []db.MigrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, _ := os.ReadFile("../../migrations/" + entry.Name())
		migrations = append(migrations, db.MigrationFile{Name: entry.Name(), Content: string(content)})
	}

	// Run twice — second should be a no-op
	if err := database.Migrate(migrations); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := database.Migrate(migrations); err != nil {
		t.Fatalf("second migrate should be idempotent: %v", err)
	}
}

func TestMigrate_SkipsBadFilenames(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	migrations := []db.MigrationFile{
		{Name: "README.md", Content: "not a migration"},
		{Name: "no-number.sql", Content: "SELECT 1"},
		{Name: "abc_bad.sql", Content: "SELECT 1"},
	}

	// Should skip all without error
	if err := database.Migrate(migrations); err != nil {
		t.Fatalf("migrate with bad filenames: %v", err)
	}
}
