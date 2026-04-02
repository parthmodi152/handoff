package db_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hitl-sh/handoff-server/internal/models"
)

func TestGetPendingRequestsByLoop_Basic(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Pending Loop")

	seedRequest(t, database, loopID, userID)
	seedRequest(t, database, loopID, userID)

	reqs, total, err := database.GetPendingRequestsByLoop(ctx, loopID, 10)
	if err != nil {
		t.Fatalf("get pending: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(reqs) != 2 {
		t.Errorf("reqs = %d, want 2", len(reqs))
	}
}

func TestGetPendingRequestsByLoop_ExcludesNonPending(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "NonPending Loop")

	reqID1 := seedRequest(t, database, loopID, userID)
	seedRequest(t, database, loopID, userID) // pending
	seedRequest(t, database, loopID, userID) // pending

	// Complete one request
	database.RespondToRequest(ctx, reqID1, userID, json.RawMessage(`{"boolean":true,"boolean_label":"Y"}`))

	reqs, total, err := database.GetPendingRequestsByLoop(ctx, loopID, 10)
	if err != nil {
		t.Fatalf("get pending: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(reqs) != 2 {
		t.Errorf("reqs = %d, want 2", len(reqs))
	}
	// None should be the completed one
	for _, r := range reqs {
		if r.ID == reqID1 {
			t.Errorf("completed request %q should not appear in pending results", reqID1)
		}
	}
}

func TestGetPendingRequestsByLoop_Limit(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Limit Loop")

	for i := 0; i < 5; i++ {
		seedRequest(t, database, loopID, userID)
	}

	reqs, total, err := database.GetPendingRequestsByLoop(ctx, loopID, 3)
	if err != nil {
		t.Fatalf("get pending: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(reqs) != 3 {
		t.Errorf("reqs = %d, want 3 (limited)", len(reqs))
	}
}

func TestGetPendingRequestsByLoop_PrioritySort(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Priority Loop")

	// Create requests with different priorities
	priorities := []string{"low", "critical", "medium", "high"}
	for _, p := range priorities {
		now := models.NowUTC()
		req := &models.Request{
			ID:             uuid.New().String(),
			LoopID:         loopID,
			CreatorID:      userID,
			ProcessingType: "time-sensitive",
			ContentType:    "markdown",
			Priority:       p,
			Title:          p + " priority title",
			RequestText:    p + " priority request",
			Platform:       "api",
			ResponseType:   "boolean",
			ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
			Status:         "pending",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := database.CreateRequest(ctx, req); err != nil {
			t.Fatalf("create request: %v", err)
		}
	}

	reqs, _, err := database.GetPendingRequestsByLoop(ctx, loopID, 10)
	if err != nil {
		t.Fatalf("get pending: %v", err)
	}
	if len(reqs) != 4 {
		t.Fatalf("reqs = %d, want 4", len(reqs))
	}
	// Expected order: critical, high, medium, low
	expected := []string{"critical", "high", "medium", "low"}
	for i, exp := range expected {
		if reqs[i].Priority != exp {
			t.Errorf("reqs[%d].Priority = %q, want %q", i, reqs[i].Priority, exp)
		}
	}
}

func TestGetPendingRequestsByLoop_Empty(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Empty Pending Loop")

	reqs, total, err := database.GetPendingRequestsByLoop(ctx, loopID, 10)
	if err != nil {
		t.Fatalf("get pending: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if reqs != nil {
		t.Errorf("reqs = %v, want nil", reqs)
	}
}

func TestGetPendingRequestsByLoop_TimeoutSort(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Timeout Sort Loop")

	now := models.NowUTC()
	// All same priority, different timeouts — earlier timeout should come first
	timeout1 := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	timeout2 := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	timeoutSeconds := 3600

	for i, ta := range []*string{&timeout2, &timeout1, nil} {
		req := &models.Request{
			ID:             uuid.New().String(),
			LoopID:         loopID,
			CreatorID:      userID,
			ProcessingType: "time-sensitive",
			ContentType:    "markdown",
			Priority:       "medium",
			Title:          "Timeout Sort " + string(rune('A'+i)),
			RequestText:    "timeout sort " + string(rune('A'+i)),
			Platform:       "api",
			ResponseType:   "boolean",
			ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
			TimeoutAt:      ta,
			Status:         "pending",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if ta != nil {
			req.TimeoutSeconds = &timeoutSeconds
		}
		if err := database.CreateRequest(ctx, req); err != nil {
			t.Fatalf("create request: %v", err)
		}
	}

	reqs, _, err := database.GetPendingRequestsByLoop(ctx, loopID, 10)
	if err != nil {
		t.Fatalf("get pending: %v", err)
	}
	if len(reqs) != 3 {
		t.Fatalf("reqs = %d, want 3", len(reqs))
	}
	// First should be the earliest timeout (1h), then 2h, then nil (NULLS LAST)
	if reqs[0].TimeoutAt == nil {
		t.Error("reqs[0] should have a timeout (earliest)")
	}
	if reqs[2].TimeoutAt != nil {
		t.Error("reqs[2] should have nil timeout (NULLS LAST)")
	}
}

func TestGetPendingRequestsByLoop_LoopNamePopulated(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Named Loop")
	seedRequest(t, database, loopID, userID)

	reqs, _, err := database.GetPendingRequestsByLoop(ctx, loopID, 10)
	if err != nil {
		t.Fatalf("get pending: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("reqs = %d, want 1", len(reqs))
	}
	if reqs[0].LoopName != "Named Loop" {
		t.Errorf("loop_name = %q, want %q", reqs[0].LoopName, "Named Loop")
	}
}
