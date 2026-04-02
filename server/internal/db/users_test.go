package db_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/hitl-sh/handoff-server/internal/models"
)

func TestUpsertUser_Insert(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	user := &models.User{
		ID:            uuid.New().String(),
		Email:         "alice@example.com",
		Name:          "Alice",
		OAuthProvider: "google",
		OAuthID:       "google-123",
		CreatedAt:     models.NowUTC(),
		UpdatedAt:     models.NowUTC(),
	}
	if err := database.UpsertUser(ctx, user); err != nil {
		t.Fatalf("upsert user: %v", err)
	}

	got, err := database.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.Email != "alice@example.com" {
		t.Errorf("email = %q, want %q", got.Email, "alice@example.com")
	}
	if got.Name != "Alice" {
		t.Errorf("name = %q, want %q", got.Name, "Alice")
	}
}

func TestUpsertUser_Update(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	user := &models.User{
		ID:            uuid.New().String(),
		Email:         "bob@example.com",
		Name:          "Bob",
		OAuthProvider: "google",
		OAuthID:       "google-bob",
		CreatedAt:     models.NowUTC(),
		UpdatedAt:     models.NowUTC(),
	}
	if err := database.UpsertUser(ctx, user); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Upsert again with updated name and avatar
	avatarURL := "https://example.com/avatar.png"
	updated := &models.User{
		ID:            uuid.New().String(), // Different ID, same oauth
		Email:         "bob-new@example.com",
		Name:          "Robert",
		AvatarURL:     &avatarURL,
		OAuthProvider: "google",
		OAuthID:       "google-bob",
		CreatedAt:     models.NowUTC(),
		UpdatedAt:     models.NowUTC(),
	}
	if err := database.UpsertUser(ctx, updated); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	// Should update existing record (keyed by oauth_provider+oauth_id)
	got, err := database.GetUserByID(ctx, user.ID) // original ID is kept
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.Name != "Robert" {
		t.Errorf("name = %q, want %q", got.Name, "Robert")
	}
	if got.Email != "bob-new@example.com" {
		t.Errorf("email = %q, want %q", got.Email, "bob-new@example.com")
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	database := openTestDB(t)

	got, err := database.GetUserByID(t.Context(), "nonexistent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestGetUserByOAuth(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	user := &models.User{
		ID:            uuid.New().String(),
		Email:         "oauth@example.com",
		Name:          "OAuth User",
		OAuthProvider: "apple",
		OAuthID:       "apple-999",
		CreatedAt:     models.NowUTC(),
		UpdatedAt:     models.NowUTC(),
	}
	if err := database.UpsertUser(ctx, user); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := database.GetUserByOAuth(ctx, "apple", "apple-999")
	if err != nil {
		t.Fatalf("get by oauth: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != user.ID {
		t.Errorf("id = %q, want %q", got.ID, user.ID)
	}

	// Not found case
	got, err = database.GetUserByOAuth(ctx, "apple", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}
