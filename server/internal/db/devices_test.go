package db_test

import (
	"testing"
)

func TestUpsertDeviceToken(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")

	if err := database.UpsertDeviceToken(ctx, userID, "token-abc", "ios", "1.0.0"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	tokens, err := database.GetDeviceTokensByUser(ctx, userID)
	if err != nil {
		t.Fatalf("get tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("tokens = %d, want 1", len(tokens))
	}
	if tokens[0].Token != "token-abc" {
		t.Errorf("token = %q, want %q", tokens[0].Token, "token-abc")
	}
	if tokens[0].Platform != "ios" {
		t.Errorf("platform = %q, want %q", tokens[0].Platform, "ios")
	}
	if tokens[0].AppVersion == nil || *tokens[0].AppVersion != "1.0.0" {
		t.Errorf("app_version = %v, want %q", tokens[0].AppVersion, "1.0.0")
	}
}

func TestUpsertDeviceToken_Reassign(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	user1 := seedUser(t, database, "u1@test.com", "User1")
	user2 := seedUser(t, database, "u2@test.com", "User2")

	// Token assigned to user1
	database.UpsertDeviceToken(ctx, user1, "shared-token", "ios", "1.0.0")

	// Same token reassigned to user2 (device sold)
	database.UpsertDeviceToken(ctx, user2, "shared-token", "ios", "2.0.0")

	// User1 should have no tokens
	tokens, _ := database.GetDeviceTokensByUser(ctx, user1)
	if len(tokens) != 0 {
		t.Errorf("user1 tokens = %d, want 0 (token reassigned)", len(tokens))
	}

	// User2 should have the token
	tokens, _ = database.GetDeviceTokensByUser(ctx, user2)
	if len(tokens) != 1 {
		t.Fatalf("user2 tokens = %d, want 1", len(tokens))
	}
	if tokens[0].UserID != user2 {
		t.Errorf("user_id = %q, want %q", tokens[0].UserID, user2)
	}
}

func TestDeleteDeviceToken(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	database.UpsertDeviceToken(ctx, userID, "del-token", "ios", "")

	if err := database.DeleteDeviceToken(ctx, "del-token"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	tokens, _ := database.GetDeviceTokensByUser(ctx, userID)
	if len(tokens) != 0 {
		t.Errorf("tokens after delete = %d, want 0", len(tokens))
	}

	// Deleting nonexistent token should not error
	if err := database.DeleteDeviceToken(ctx, "nonexistent"); err != nil {
		t.Fatalf("delete nonexistent: %v", err)
	}
}

func TestGetDeviceTokensByLoopMembers(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	creator := seedUser(t, database, "creator@test.com", "Creator")
	member := seedUser(t, database, "member@test.com", "Member")
	loopID, _ := seedLoop(t, database, creator, "Token Loop")
	database.JoinLoop(ctx, loopID, member)

	database.UpsertDeviceToken(ctx, creator, "creator-token", "ios", "")
	database.UpsertDeviceToken(ctx, member, "member-token", "ios", "")

	// Get all tokens
	tokens, err := database.GetDeviceTokensByLoopMembers(ctx, loopID, "")
	if err != nil {
		t.Fatalf("get tokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("all tokens = %d, want 2", len(tokens))
	}

	// Exclude creator
	tokens, err = database.GetDeviceTokensByLoopMembers(ctx, loopID, creator)
	if err != nil {
		t.Fatalf("get tokens excluding: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("tokens excluding creator = %d, want 1", len(tokens))
	}
	if tokens[0].UserID != member {
		t.Errorf("token user_id = %q, want %q", tokens[0].UserID, member)
	}
}

func TestGetPendingCountForUser(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Count Loop")

	seedRequest(t, database, loopID, userID)
	seedRequest(t, database, loopID, userID)

	count, err := database.GetPendingCountForUser(ctx, userID)
	if err != nil {
		t.Fatalf("get pending count: %v", err)
	}
	if count != 2 {
		t.Errorf("pending count = %d, want 2", count)
	}
}

func TestDeleteDeviceTokensByUser(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	database.UpsertDeviceToken(ctx, userID, "token1", "ios", "")
	database.UpsertDeviceToken(ctx, userID, "token2", "ios", "")

	if err := database.DeleteDeviceTokensByUser(ctx, userID); err != nil {
		t.Fatalf("delete all: %v", err)
	}

	tokens, _ := database.GetDeviceTokensByUser(ctx, userID)
	if len(tokens) != 0 {
		t.Errorf("tokens after delete all = %d, want 0", len(tokens))
	}
}
