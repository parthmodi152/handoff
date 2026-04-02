package db_test

import (
	"testing"
)

func TestUpsertActivityToken(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Activity Loop")

	if err := database.UpsertActivityToken(ctx, userID, loopID, "activity-token-abc"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	tokens, err := database.GetActivityTokensByLoop(ctx, loopID)
	if err != nil {
		t.Fatalf("get tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("tokens = %d, want 1", len(tokens))
	}
	if tokens[0].Token != "activity-token-abc" {
		t.Errorf("token = %q, want %q", tokens[0].Token, "activity-token-abc")
	}
	if tokens[0].UserID != userID {
		t.Errorf("user_id = %q, want %q", tokens[0].UserID, userID)
	}
	if tokens[0].LoopID != loopID {
		t.Errorf("loop_id = %q, want %q", tokens[0].LoopID, loopID)
	}
}

func TestUpsertActivityToken_Reassign(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	user1 := seedUser(t, database, "u1@test.com", "User1")
	user2 := seedUser(t, database, "u2@test.com", "User2")
	loop1, _ := seedLoop(t, database, user1, "Loop1")
	loop2, _ := seedLoop(t, database, user2, "Loop2")

	// Token assigned to user1/loop1
	database.UpsertActivityToken(ctx, user1, loop1, "shared-activity-token")

	// Same token reassigned to user2/loop2
	database.UpsertActivityToken(ctx, user2, loop2, "shared-activity-token")

	// Loop1 should have no tokens
	tokens, _ := database.GetActivityTokensByLoop(ctx, loop1)
	if len(tokens) != 0 {
		t.Errorf("loop1 tokens = %d, want 0 (token reassigned)", len(tokens))
	}

	// Loop2 should have the token
	tokens, _ = database.GetActivityTokensByLoop(ctx, loop2)
	if len(tokens) != 1 {
		t.Fatalf("loop2 tokens = %d, want 1", len(tokens))
	}
	if tokens[0].UserID != user2 {
		t.Errorf("user_id = %q, want %q", tokens[0].UserID, user2)
	}
}

func TestDeleteActivityToken(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Del Activity Loop")
	database.UpsertActivityToken(ctx, userID, loopID, "del-activity-token")

	if err := database.DeleteActivityToken(ctx, "del-activity-token"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	tokens, _ := database.GetActivityTokensByLoop(ctx, loopID)
	if len(tokens) != 0 {
		t.Errorf("tokens after delete = %d, want 0", len(tokens))
	}

	// Deleting nonexistent token should not error
	if err := database.DeleteActivityToken(ctx, "nonexistent"); err != nil {
		t.Fatalf("delete nonexistent: %v", err)
	}
}

func TestGetActivityTokensByLoop_MultipleTokens(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	user1 := seedUser(t, database, "u1@test.com", "User1")
	user2 := seedUser(t, database, "u2@test.com", "User2")
	loopID, _ := seedLoop(t, database, user1, "Multi Token Loop")
	database.JoinLoop(ctx, loopID, user2)

	database.UpsertActivityToken(ctx, user1, loopID, "activity-token-1")
	database.UpsertActivityToken(ctx, user2, loopID, "activity-token-2")

	tokens, err := database.GetActivityTokensByLoop(ctx, loopID)
	if err != nil {
		t.Fatalf("get tokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("tokens = %d, want 2", len(tokens))
	}
}

func TestGetActivityTokensByLoop_EmptyLoop(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Empty Loop")

	tokens, err := database.GetActivityTokensByLoop(ctx, loopID)
	if err != nil {
		t.Fatalf("get tokens: %v", err)
	}
	if tokens != nil {
		t.Errorf("tokens = %v, want nil", tokens)
	}
}

func TestDeleteActivityTokensByUser(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loop1, _ := seedLoop(t, database, userID, "Loop1")
	loop2, _ := seedLoop(t, database, userID, "Loop2")

	database.UpsertActivityToken(ctx, userID, loop1, "token-1")
	database.UpsertActivityToken(ctx, userID, loop2, "token-2")

	if err := database.DeleteActivityTokensByUser(ctx, userID); err != nil {
		t.Fatalf("delete all: %v", err)
	}

	tokens1, _ := database.GetActivityTokensByLoop(ctx, loop1)
	tokens2, _ := database.GetActivityTokensByLoop(ctx, loop2)
	if len(tokens1)+len(tokens2) != 0 {
		t.Errorf("tokens after delete all = %d, want 0", len(tokens1)+len(tokens2))
	}
}
