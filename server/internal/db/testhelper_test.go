package db_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/models"
)

// openTestDB creates a temporary SQLite database with all migrations applied.
func openTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var migrations []db.MigrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile("../../migrations/" + entry.Name())
		if err != nil {
			t.Fatalf("read migration %s: %v", entry.Name(), err)
		}
		migrations = append(migrations, db.MigrationFile{Name: entry.Name(), Content: string(content)})
	}
	if err := database.Migrate(migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Cleanup(func() { database.Close() })
	return database
}

// seedUser inserts a user directly and returns their ID.
func seedUser(t *testing.T, database *db.DB, email, name string) string {
	t.Helper()
	id := uuid.New().String()
	now := models.NowUTC()
	_, err := database.Exec(
		`INSERT INTO users (id, email, name, oauth_provider, oauth_id, created_at, updated_at)
		 VALUES (?, ?, ?, 'dev', ?, ?, ?)`,
		id, email, name, email, now, now,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

// seedLoop creates a loop and adds the creator as member. Returns (loopID, inviteCode).
func seedLoop(t *testing.T, database *db.DB, creatorID, name string) (string, string) {
	t.Helper()
	id := uuid.New().String()
	inviteCode := uuid.New().String()[:8]
	now := models.NowUTC()
	loop := &models.Loop{
		ID:         id,
		Name:       name,
		Description: "test loop",
		Icon:       "test",
		CreatorID:  creatorID,
		InviteCode: inviteCode,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := database.CreateLoop(t.Context(), loop); err != nil {
		t.Fatalf("seed loop: %v", err)
	}
	return id, inviteCode
}

// seedRequest creates a pending boolean request. Returns requestID.
func seedRequest(t *testing.T, database *db.DB, loopID, creatorID string) string {
	t.Helper()
	id := uuid.New().String()
	now := models.NowUTC()
	req := &models.Request{
		ID:             id,
		LoopID:         loopID,
		CreatorID:      creatorID,
		ProcessingType: "time-sensitive",
		ContentType:    "markdown",
		Priority:       "medium",
		Title:          "Test Title",
		RequestText:    "Test request",
		Platform:       "api",
		ResponseType:   "boolean",
		ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := database.CreateRequest(t.Context(), req); err != nil {
		t.Fatalf("seed request: %v", err)
	}
	return id
}
