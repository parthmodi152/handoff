package db_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hitl-sh/handoff-server/internal/models"
)

func TestCreateRequest(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Req Loop")
	reqID := seedRequest(t, database, loopID, userID)

	req, err := database.GetRequest(ctx, reqID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if req == nil {
		t.Fatal("expected request, got nil")
	}
	if req.Status != "pending" {
		t.Errorf("status = %q, want %q", req.Status, "pending")
	}
	if req.ResponseType != "boolean" {
		t.Errorf("response_type = %q, want %q", req.ResponseType, "boolean")
	}
	if req.LoopName != "Req Loop" {
		t.Errorf("loop_name = %q, want %q", req.LoopName, "Req Loop")
	}
}

func TestGetRequest_NotFound(t *testing.T) {
	database := openTestDB(t)

	req, err := database.GetRequest(t.Context(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req != nil {
		t.Fatalf("expected nil, got %+v", req)
	}
}

func TestGetRequest_JSONRoundTrip(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "JSON Loop")

	now := models.NowUTC()
	contextData := json.RawMessage(`{"agent":"test-agent","version":2}`)
	responseConfig := json.RawMessage(`{"true_label":"Approve","false_label":"Reject"}`)
	defaultResp := json.RawMessage(`{"boolean":false,"boolean_label":"Reject"}`)

	req := &models.Request{
		ID:              uuid.New().String(),
		LoopID:          loopID,
		CreatorID:       userID,
		ProcessingType:  "time-sensitive",
		ContentType:     "markdown",
		Priority:        "high",
		Title:           "JSON Test Title",
		RequestText:     "JSON test",
		Platform:        "api",
		ResponseType:    "boolean",
		ResponseConfig:  responseConfig,
		Context:         contextData,
		DefaultResponse: defaultResp,
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := database.CreateRequest(ctx, req); err != nil {
		t.Fatalf("create request: %v", err)
	}

	got, err := database.GetRequest(ctx, req.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}

	// Verify JSON fields survived round-trip
	var ctxMap map[string]interface{}
	if err := json.Unmarshal(got.Context, &ctxMap); err != nil {
		t.Fatalf("unmarshal context: %v", err)
	}
	if ctxMap["agent"] != "test-agent" {
		t.Errorf("context.agent = %v, want %q", ctxMap["agent"], "test-agent")
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(got.ResponseConfig, &configMap); err != nil {
		t.Fatalf("unmarshal response_config: %v", err)
	}
	if configMap["true_label"] != "Approve" {
		t.Errorf("response_config.true_label = %v, want %q", configMap["true_label"], "Approve")
	}

	var defaultMap map[string]interface{}
	if err := json.Unmarshal(got.DefaultResponse, &defaultMap); err != nil {
		t.Fatalf("unmarshal default_response: %v", err)
	}
	if defaultMap["boolean"] != false {
		t.Errorf("default_response.boolean = %v, want false", defaultMap["boolean"])
	}
}

func TestListRequests_Filters(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Filter Loop")

	// Create 3 requests with different priorities
	for _, priority := range []string{"low", "medium", "high"} {
		now := models.NowUTC()
		req := &models.Request{
			ID:             uuid.New().String(),
			LoopID:         loopID,
			CreatorID:      userID,
			ProcessingType: "time-sensitive",
			ContentType:    "markdown",
			Priority:       priority,
			Title:          priority + " title",
			RequestText:    priority + " request",
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
		// Small delay so created_at differs
		time.Sleep(10 * time.Millisecond)
	}

	// Filter by priority
	results, total, err := database.ListRequests(ctx, userID, "", "high", "", "", 50, 0)
	if err != nil {
		t.Fatalf("list by priority: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Priority != "high" {
		t.Errorf("priority = %q, want %q", results[0].Priority, "high")
	}

	// Filter by status — all are pending
	results, total, err = database.ListRequests(ctx, userID, "pending", "", "", "", 50, 0)
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}

	// Filter by loop
	results, total, err = database.ListRequests(ctx, userID, "", "", loopID, "", 50, 0)
	if err != nil {
		t.Fatalf("list by loop: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}

	// No results for completed status
	results, total, err = database.ListRequests(ctx, userID, "completed", "", "", "", 50, 0)
	if err != nil {
		t.Fatalf("list completed: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

func TestListRequests_Sort(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Sort Loop")

	// Use explicit timestamps with second-level differences for reliable ordering
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	priorities := []string{"low", "high", "medium"}
	for i, p := range priorities {
		ts := baseTime.Add(time.Duration(i) * time.Second).Format(time.RFC3339)
		req := &models.Request{
			ID:             uuid.New().String(),
			LoopID:         loopID,
			CreatorID:      userID,
			ProcessingType: "time-sensitive",
			ContentType:    "markdown",
			Priority:       p,
			Title:          p + " title",
			RequestText:    p + " request",
			Platform:       "api",
			ResponseType:   "boolean",
			ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
			Status:         "pending",
			CreatedAt:      ts,
			UpdatedAt:      ts,
		}
		database.CreateRequest(ctx, req)
	}

	// Default sort: created_at DESC (newest first = "medium" at +2s)
	results, _, err := database.ListRequests(ctx, userID, "", "", "", "", 50, 0)
	if err != nil {
		t.Fatalf("list default sort: %v", err)
	}
	if len(results) < 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Priority != "medium" {
		t.Errorf("default sort first = %q, want %q", results[0].Priority, "medium")
	}

	// Sort: created_at_asc (oldest first = "low" at +0s)
	results, _, err = database.ListRequests(ctx, userID, "", "", "", "created_at_asc", 50, 0)
	if err != nil {
		t.Fatalf("list asc sort: %v", err)
	}
	if results[0].Priority != "low" {
		t.Errorf("asc sort first = %q, want %q", results[0].Priority, "low")
	}

	// Sort: priority_desc (highest priority first = "high")
	results, _, err = database.ListRequests(ctx, userID, "", "", "", "priority_desc", 50, 0)
	if err != nil {
		t.Fatalf("list priority sort: %v", err)
	}
	if results[0].Priority != "high" {
		t.Errorf("priority sort first = %q, want %q", results[0].Priority, "high")
	}
}

func TestListRequests_Pagination(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Page Loop")

	// Create 5 requests
	for i := 0; i < 5; i++ {
		now := models.NowUTC()
		req := &models.Request{
			ID:             uuid.New().String(),
			LoopID:         loopID,
			CreatorID:      userID,
			ProcessingType: "time-sensitive",
			ContentType:    "markdown",
			Priority:       "medium",
			Title:          "Test Title",
			RequestText:    "request",
			Platform:       "api",
			ResponseType:   "boolean",
			ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
			Status:         "pending",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		database.CreateRequest(ctx, req)
		time.Sleep(5 * time.Millisecond)
	}

	// Page 1: limit 2, offset 0
	results, total, err := database.ListRequests(ctx, userID, "", "", "", "", 2, 0)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(results) != 2 {
		t.Errorf("page 1 results = %d, want 2", len(results))
	}

	// Page 2: limit 2, offset 2
	results, total, err = database.ListRequests(ctx, userID, "", "", "", "", 2, 2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(results) != 2 {
		t.Errorf("page 2 results = %d, want 2", len(results))
	}

	// Page 3: limit 2, offset 4 — only 1 left
	results, _, err = database.ListRequests(ctx, userID, "", "", "", "", 2, 4)
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("page 3 results = %d, want 1", len(results))
	}
}

func TestListRequests_MembershipFilter(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	user1 := seedUser(t, database, "u1@test.com", "User1")
	user2 := seedUser(t, database, "u2@test.com", "User2")
	loop1, _ := seedLoop(t, database, user1, "Loop1")
	loop2, _ := seedLoop(t, database, user2, "Loop2")

	seedRequest(t, database, loop1, user1)
	seedRequest(t, database, loop2, user2)

	// User1 should only see requests from loop1
	results, total, _ := database.ListRequests(ctx, user1, "", "", "", "", 50, 0)
	if total != 1 {
		t.Errorf("user1 total = %d, want 1", total)
	}
	if len(results) > 0 && results[0].LoopID != loop1 {
		t.Errorf("user1 sees wrong loop: %q", results[0].LoopID)
	}
}

func TestRespondToRequest(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Resp Loop")
	reqID := seedRequest(t, database, loopID, userID)

	respData := json.RawMessage(`{"boolean":true,"boolean_label":"Y"}`)
	if err := database.RespondToRequest(ctx, reqID, userID, respData); err != nil {
		t.Fatalf("respond: %v", err)
	}

	req, err := database.GetRequest(ctx, reqID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if req.Status != "completed" {
		t.Errorf("status = %q, want %q", req.Status, "completed")
	}
	if req.ResponseBy == nil || *req.ResponseBy != userID {
		t.Errorf("response_by = %v, want %q", req.ResponseBy, userID)
	}
	if req.ResponseAt == nil {
		t.Error("response_at should not be nil")
	}
	if req.ResponseTimeSeconds == nil {
		t.Error("response_time_seconds should not be nil")
	}

	var data map[string]interface{}
	json.Unmarshal(req.ResponseData, &data)
	if data["boolean"] != true {
		t.Errorf("response_data.boolean = %v, want true", data["boolean"])
	}
}

func TestRespondToRequest_NotPending(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "NP Loop")
	reqID := seedRequest(t, database, loopID, userID)

	// Cancel the request first
	database.CancelRequest(ctx, reqID)

	// Attempt to respond — the WHERE clause has status='pending', so this is a no-op
	respData := json.RawMessage(`{"boolean":true,"boolean_label":"Y"}`)
	if err := database.RespondToRequest(ctx, reqID, userID, respData); err != nil {
		t.Fatalf("respond: %v", err)
	}

	// Should still be cancelled
	req, _ := database.GetRequest(ctx, reqID)
	if req.Status != "cancelled" {
		t.Errorf("status = %q, want %q", req.Status, "cancelled")
	}
}

func TestCancelRequest(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Cancel Loop")
	reqID := seedRequest(t, database, loopID, userID)

	if err := database.CancelRequest(ctx, reqID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	req, _ := database.GetRequest(ctx, reqID)
	if req.Status != "cancelled" {
		t.Errorf("status = %q, want %q", req.Status, "cancelled")
	}
}

func TestExpireTimedOutRequests(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Timeout Loop")

	// Create a request with timeout in the past
	pastTimeout := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	now := models.NowUTC()
	timeoutSeconds := 60
	req := &models.Request{
		ID:             uuid.New().String(),
		LoopID:         loopID,
		CreatorID:      userID,
		ProcessingType: "time-sensitive",
		ContentType:    "markdown",
		Priority:       "medium",
		Title:          "Timeout Test Title",
		RequestText:    "timeout test",
		Platform:       "api",
		ResponseType:   "boolean",
		ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
		TimeoutSeconds: &timeoutSeconds,
		TimeoutAt:      &pastTimeout,
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := database.CreateRequest(ctx, req); err != nil {
		t.Fatalf("create: %v", err)
	}

	expired, err := database.ExpireTimedOutRequests(ctx)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired = %d, want 1", len(expired))
	}
	if expired[0].ID != req.ID {
		t.Errorf("expired id = %q, want %q", expired[0].ID, req.ID)
	}

	// Verify DB state is updated
	got, _ := database.GetRequest(ctx, req.ID)
	if got.Status != "timeout" {
		t.Errorf("db status = %q, want %q", got.Status, "timeout")
	}
}

func TestExpireTimedOutRequests_NoTimeout(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "No Timeout Loop")

	// Request without timeout_at should never expire
	seedRequest(t, database, loopID, userID)

	expired, err := database.ExpireTimedOutRequests(ctx)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if len(expired) != 0 {
		t.Errorf("expired = %d, want 0", len(expired))
	}
}

func TestExpireTimedOutRequests_IgnoresCompleted(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "Completed Loop")

	// Create request with past timeout, then complete it
	pastTimeout := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	now := models.NowUTC()
	timeoutSeconds := 60
	req := &models.Request{
		ID:             uuid.New().String(),
		LoopID:         loopID,
		CreatorID:      userID,
		ProcessingType: "time-sensitive",
		ContentType:    "markdown",
		Priority:       "medium",
		Title:          "Completed Before Timeout",
		RequestText:    "completed before timeout",
		Platform:       "api",
		ResponseType:   "boolean",
		ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
		TimeoutSeconds: &timeoutSeconds,
		TimeoutAt:      &pastTimeout,
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	database.CreateRequest(ctx, req)
	database.RespondToRequest(ctx, req.ID, userID, json.RawMessage(`{"boolean":true,"boolean_label":"Y"}`))

	expired, err := database.ExpireTimedOutRequests(ctx)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if len(expired) != 0 {
		t.Errorf("expired = %d, want 0 (already completed)", len(expired))
	}
}

// TestScanRequest_TimeoutOnRead verifies the scanRequest behavior where
// it silently changes status to "timeout" on read for requests past their
// timeout_at. This only affects row-scanning paths (ListRequests, GetRecentRequests),
// NOT GetRequest which has its own inline scanning without this mutation.
//
// This is a known design inconsistency:
// - GetRequest returns status="pending" for a timed-out request
// - ListRequests returns status="timeout" for the same request (via scanRequest)
// - The DB retains status="pending" until the timeout worker runs
func TestScanRequest_TimeoutOnRead(t *testing.T) {
	database := openTestDB(t)
	ctx := t.Context()

	userID := seedUser(t, database, "u@test.com", "User")
	loopID, _ := seedLoop(t, database, userID, "ScanTimeout Loop")

	pastTimeout := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	now := models.NowUTC()
	timeoutSeconds := 60
	defaultResp := json.RawMessage(`{"boolean":false,"boolean_label":"N"}`)
	req := &models.Request{
		ID:              uuid.New().String(),
		LoopID:          loopID,
		CreatorID:       userID,
		ProcessingType:  "time-sensitive",
		ContentType:     "markdown",
		Priority:        "medium",
		Title:           "Scan Timeout Test",
		RequestText:     "scan timeout test",
		Platform:        "api",
		ResponseType:    "boolean",
		ResponseConfig:  json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
		DefaultResponse: defaultResp,
		TimeoutSeconds:  &timeoutSeconds,
		TimeoutAt:       &pastTimeout,
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	database.CreateRequest(ctx, req)

	// GetRequest has inline scanning — does NOT have timeout-on-read mutation
	gotDirect, err := database.GetRequest(ctx, req.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if gotDirect.Status != "pending" {
		t.Errorf("GetRequest status = %q, want %q (no mutation in GetRequest)", gotDirect.Status, "pending")
	}

	// ListRequests uses scanRequest which DOES mutate status on read
	results, _, err := database.ListRequests(ctx, userID, "pending", "", "", "", 50, 0)
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}

	// Find our request in the results
	var gotFromList *models.Request
	for i := range results {
		if results[i].ID == req.ID {
			gotFromList = &results[i]
			break
		}
	}
	if gotFromList == nil {
		t.Fatal("request not found in list results")
	}

	// BUG: scanRequest mutates status to "timeout" on read but DB still has "pending"
	if gotFromList.Status != "timeout" {
		t.Errorf("ListRequests status = %q, want %q (scanRequest should mutate)", gotFromList.Status, "timeout")
	}

	// BUG: DefaultResponse should be copied to ResponseData
	if gotFromList.ResponseData == nil {
		t.Error("expected DefaultResponse to be copied to ResponseData by scanRequest")
	} else {
		var data map[string]interface{}
		json.Unmarshal(gotFromList.ResponseData, &data)
		if data["boolean"] != false {
			t.Errorf("response_data.boolean = %v, want false", data["boolean"])
		}
	}

	// INCONSISTENCY: GetRequest and ListRequests return different statuses for the same request
	t.Log("BUG CONFIRMED: GetRequest returns 'pending' but ListRequests returns 'timeout' for the same " +
		"request. scanRequest mutates status in memory without DB update. The timeout worker may " +
		"re-process this request, and agents get inconsistent responses depending on which endpoint they call.")
}
