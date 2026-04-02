package db_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/models"
)

// seedAPIKey creates an API key directly and returns (keyID, rawKey).
func seedAPIKey(t *testing.T, database *db.DB, userID string, opts ...func(*models.APIKey)) (string, string) {
	t.Helper()
	raw := "hnd_" + uuid.New().String()[:16]
	hash, _ := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.MinCost)
	key := &models.APIKey{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        "test-key",
		KeyHash:     string(hash),
		KeyPrefix:   raw[:8],
		Permissions: []string{"requests:read", "requests:write", "loops:read", "loops:write"},
		IsActive:    true,
		CreatedAt:   models.NowUTC(),
	}
	for _, o := range opts {
		o(key)
	}
	if err := database.CreateAPIKey(context.Background(), key); err != nil {
		t.Fatalf("seed api key: %v", err)
	}
	return key.ID, raw
}

func TestCreateAPIKey_Basic(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()
	userID := seedUser(t, database, "ak@test.com", "AK User")

	keyID, _ := seedAPIKey(t, database, userID)

	keys, err := database.ListAPIKeys(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].ID != keyID {
		t.Errorf("id = %q, want %q", keys[0].ID, keyID)
	}
	if keys[0].AgentName != nil {
		t.Errorf("agent_name = %v, want nil", keys[0].AgentName)
	}
	if len(keys[0].AllowedLoops) != 0 {
		t.Errorf("allowed_loops = %v, want empty", keys[0].AllowedLoops)
	}
}

func TestCreateAPIKey_WithAgentName(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()
	userID := seedUser(t, database, "agent@test.com", "Agent User")

	agentName := "Claude"
	seedAPIKey(t, database, userID, func(k *models.APIKey) {
		k.AgentName = &agentName
	})

	keys, err := database.ListAPIKeys(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].AgentName == nil {
		t.Fatal("agent_name = nil, want 'Claude'")
	}
	if *keys[0].AgentName != "Claude" {
		t.Errorf("agent_name = %q, want %q", *keys[0].AgentName, "Claude")
	}
}

func TestCreateAPIKey_WithAllowedLoops(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()
	userID := seedUser(t, database, "scoped@test.com", "Scoped User")
	loop1, _ := seedLoop(t, database, userID, "Loop A")
	loop2, _ := seedLoop(t, database, userID, "Loop B")
	seedLoop(t, database, userID, "Loop C") // not allowed

	seedAPIKey(t, database, userID, func(k *models.APIKey) {
		k.AllowedLoops = []string{loop1, loop2}
	})

	keys, err := database.ListAPIKeys(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if len(keys[0].AllowedLoops) != 2 {
		t.Fatalf("allowed_loops = %d, want 2", len(keys[0].AllowedLoops))
	}

	// Check the loops are the ones we set
	allowed := make(map[string]bool)
	for _, id := range keys[0].AllowedLoops {
		allowed[id] = true
	}
	if !allowed[loop1] || !allowed[loop2] {
		t.Errorf("allowed_loops = %v, want [%s, %s]", keys[0].AllowedLoops, loop1, loop2)
	}
}

func TestGetAPIKeysByPrefix_LoadsAllowedLoops(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()
	userID := seedUser(t, database, "prefix@test.com", "Prefix User")
	loopID, _ := seedLoop(t, database, userID, "PrefixLoop")

	_, rawKey := seedAPIKey(t, database, userID, func(k *models.APIKey) {
		k.AllowedLoops = []string{loopID}
	})

	prefix := rawKey[:8]
	keys, err := database.GetAPIKeysByPrefix(ctx, prefix)
	if err != nil {
		t.Fatalf("get by prefix: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if len(keys[0].AllowedLoops) != 1 || keys[0].AllowedLoops[0] != loopID {
		t.Errorf("allowed_loops = %v, want [%s]", keys[0].AllowedLoops, loopID)
	}
}

func TestGetAPIKeysByPrefix_LoadsAgentName(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()
	userID := seedUser(t, database, "prefix-agent@test.com", "Prefix Agent")

	agent := "DeployBot"
	_, rawKey := seedAPIKey(t, database, userID, func(k *models.APIKey) {
		k.AgentName = &agent
	})

	prefix := rawKey[:8]
	keys, err := database.GetAPIKeysByPrefix(ctx, prefix)
	if err != nil {
		t.Fatalf("get by prefix: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].AgentName == nil || *keys[0].AgentName != "DeployBot" {
		t.Errorf("agent_name = %v, want 'DeployBot'", keys[0].AgentName)
	}
}

func TestCreateRequest_WithAgentName(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()
	userID := seedUser(t, database, "req-agent@test.com", "Req Agent")
	loopID, _ := seedLoop(t, database, userID, "Agent Loop")

	agent := "MyBot"
	now := models.NowUTC()
	req := &models.Request{
		ID:             uuid.New().String(),
		LoopID:         loopID,
		CreatorID:      userID,
		AgentName:      &agent,
		ProcessingType: "time-sensitive",
		ContentType:    "markdown",
		Priority:       "medium",
		Title:          "Agent Request Title",
		RequestText:    "Agent request",
		Platform:       "api",
		ResponseType:   "boolean",
		ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := database.CreateRequest(ctx, req); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := database.GetRequest(ctx, req.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AgentName == nil {
		t.Fatal("agent_name = nil, want 'MyBot'")
	}
	if *got.AgentName != "MyBot" {
		t.Errorf("agent_name = %q, want %q", *got.AgentName, "MyBot")
	}
}

func TestCreateRequest_WithoutAgentName(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()
	userID := seedUser(t, database, "req-noagent@test.com", "No Agent")
	loopID, _ := seedLoop(t, database, userID, "NoAgent Loop")

	reqID := seedRequest(t, database, loopID, userID)

	got, err := database.GetRequest(ctx, reqID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AgentName != nil {
		t.Errorf("agent_name = %v, want nil", got.AgentName)
	}
}

func TestListRequests_AgentNameRoundTrip(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()
	userID := seedUser(t, database, "list-agent@test.com", "List Agent")
	loopID, _ := seedLoop(t, database, userID, "ListAgent Loop")

	agent := "Claude"
	now := models.NowUTC()
	req := &models.Request{
		ID:             uuid.New().String(),
		LoopID:         loopID,
		CreatorID:      userID,
		AgentName:      &agent,
		ProcessingType: "time-sensitive",
		ContentType:    "markdown",
		Priority:       "medium",
		Title:          "List Agent Test Title",
		RequestText:    "list agent test",
		Platform:       "api",
		ResponseType:   "boolean",
		ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	database.CreateRequest(ctx, req)

	results, _, err := database.ListRequests(ctx, userID, "", "", "", "", 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].AgentName == nil || *results[0].AgentName != "Claude" {
		t.Errorf("agent_name = %v, want 'Claude'", results[0].AgentName)
	}
}
