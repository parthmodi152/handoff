package db_test

import (
	"strings"
	"testing"
)

func TestCreateLoop_AddsCreatorAsMember(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "creator@test.com", "Creator")
	loopID, _ := seedLoop(t, database, userID, "Test Loop")

	// Creator should be a member with 'creator' role
	isMember, role, err := database.IsLoopMember(ctx, loopID, userID)
	if err != nil {
		t.Fatalf("is loop member: %v", err)
	}
	if !isMember {
		t.Fatal("creator should be a member")
	}
	if role != "creator" {
		t.Errorf("role = %q, want %q", role, "creator")
	}
}

func TestGetLoop(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, inviteCode := seedLoop(t, database, userID, "My Loop")

	loop, err := database.GetLoop(ctx, loopID)
	if err != nil {
		t.Fatalf("get loop: %v", err)
	}
	if loop == nil {
		t.Fatal("expected loop, got nil")
	}
	if loop.Name != "My Loop" {
		t.Errorf("name = %q, want %q", loop.Name, "My Loop")
	}
	if loop.InviteCode != inviteCode {
		t.Errorf("invite_code = %q, want %q", loop.InviteCode, inviteCode)
	}
	if loop.CreatorID != userID {
		t.Errorf("creator_id = %q, want %q", loop.CreatorID, userID)
	}
}

func TestGetLoop_NotFound(t *testing.T) {
	database := openTestDB(t)

	loop, err := database.GetLoop(t.Context(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loop != nil {
		t.Fatalf("expected nil, got %+v", loop)
	}
}

func TestGetLoopByInviteCode(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, inviteCode := seedLoop(t, database, userID, "Invite Loop")

	// Exact match
	loop, err := database.GetLoopByInviteCode(ctx, inviteCode)
	if err != nil {
		t.Fatalf("get by invite code: %v", err)
	}
	if loop == nil || loop.ID != loopID {
		t.Fatalf("expected loop %s, got %+v", loopID, loop)
	}

	// Case-insensitive match
	loop, err = database.GetLoopByInviteCode(ctx, strings.ToUpper(inviteCode))
	if err != nil {
		t.Fatalf("case insensitive: %v", err)
	}
	if loop == nil || loop.ID != loopID {
		t.Fatalf("expected case-insensitive match, got %+v", loop)
	}

	// Not found
	loop, err = database.GetLoopByInviteCode(ctx, "ZZZZZZZZ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loop != nil {
		t.Fatalf("expected nil, got %+v", loop)
	}
}

func TestJoinLoop(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	creator := seedUser(t, database, "creator@test.com", "Creator")
	joiner := seedUser(t, database, "joiner@test.com", "Joiner")
	loopID, _ := seedLoop(t, database, creator, "Join Loop")

	// Joiner is not yet a member
	isMember, _, _ := database.IsLoopMember(ctx, loopID, joiner)
	if isMember {
		t.Fatal("joiner should not be a member yet")
	}

	// Join
	if err := database.JoinLoop(ctx, loopID, joiner); err != nil {
		t.Fatalf("join loop: %v", err)
	}

	// Now a member with 'member' role
	isMember, role, err := database.IsLoopMember(ctx, loopID, joiner)
	if err != nil {
		t.Fatalf("is loop member: %v", err)
	}
	if !isMember {
		t.Fatal("joiner should be a member after join")
	}
	if role != "member" {
		t.Errorf("role = %q, want %q", role, "member")
	}
}

func TestJoinLoop_Duplicate(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	creator := seedUser(t, database, "creator@test.com", "Creator")
	joiner := seedUser(t, database, "joiner@test.com", "Joiner")
	loopID, _ := seedLoop(t, database, creator, "Dup Loop")

	if err := database.JoinLoop(ctx, loopID, joiner); err != nil {
		t.Fatalf("first join: %v", err)
	}

	// Second join should fail (UNIQUE constraint on loop_id, user_id)
	err := database.JoinLoop(ctx, loopID, joiner)
	if err == nil {
		t.Fatal("expected error on duplicate join, got nil")
	}
}

func TestListLoops_OnlyMemberLoops(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	user1 := seedUser(t, database, "u1@test.com", "User1")
	user2 := seedUser(t, database, "u2@test.com", "User2")

	seedLoop(t, database, user1, "Loop A")
	seedLoop(t, database, user1, "Loop B")
	seedLoop(t, database, user2, "Loop C") // user2's loop

	loops, err := database.ListLoops(ctx, user1)
	if err != nil {
		t.Fatalf("list loops: %v", err)
	}
	if len(loops) != 2 {
		t.Fatalf("expected 2 loops, got %d", len(loops))
	}

	// User2 only sees their own loop
	loops, err = database.ListLoops(ctx, user2)
	if err != nil {
		t.Fatalf("list loops user2: %v", err)
	}
	if len(loops) != 1 {
		t.Fatalf("expected 1 loop, got %d", len(loops))
	}
	if loops[0].Name != "Loop C" {
		t.Errorf("loop name = %q, want %q", loops[0].Name, "Loop C")
	}
}

func TestGetLoopMembers(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	creator := seedUser(t, database, "creator@test.com", "Creator")
	member := seedUser(t, database, "member@test.com", "Member")
	loopID, _ := seedLoop(t, database, creator, "Members Loop")
	database.JoinLoop(ctx, loopID, member)

	members, err := database.GetLoopMembers(ctx, loopID)
	if err != nil {
		t.Fatalf("get members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestGetLoopMemberCount(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	creator := seedUser(t, database, "creator@test.com", "Creator")
	loopID, _ := seedLoop(t, database, creator, "Count Loop")

	count, err := database.GetLoopMemberCount(ctx, loopID)
	if err != nil {
		t.Fatalf("get count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	member := seedUser(t, database, "m@test.com", "Member")
	database.JoinLoop(ctx, loopID, member)

	count, err = database.GetLoopMemberCount(ctx, loopID)
	if err != nil {
		t.Fatalf("get count after join: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestInviteCodeExists(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	_, inviteCode := seedLoop(t, database, userID, "Code Loop")

	exists, err := database.InviteCodeExists(ctx, inviteCode)
	if err != nil {
		t.Fatalf("invite code exists: %v", err)
	}
	if !exists {
		t.Fatal("expected invite code to exist")
	}

	exists, err = database.InviteCodeExists(ctx, "NONEXIST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected invite code to not exist")
	}
}
