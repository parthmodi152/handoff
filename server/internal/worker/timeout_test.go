package worker

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sideshow/apns2/payload"

	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/events"
	"github.com/hitl-sh/handoff-server/internal/models"
)

// mockPushSender records calls to NotifyLoopMembers and live activity methods.
type mockPushSender struct {
	calls             []pushCall
	updateActivityIDs []string
	endActivityIDs    []string
}

type pushCall struct {
	LoopID        string
	ExcludeUserID string
}

func (m *mockPushSender) NotifyLoopMembers(_ context.Context, loopID, excludeUserID string, _ *payload.Payload) (int, int) {
	m.calls = append(m.calls, pushCall{LoopID: loopID, ExcludeUserID: excludeUserID})
	return 1, 0
}

func (m *mockPushSender) NotifyUser(_ context.Context, _ string, _ *payload.Payload) (int, int) {
	return 0, 0
}

func (m *mockPushSender) UpdateLoopActivity(_ context.Context, loopID string, _ interface{}, _ *int64, _, _ string) (int, int) {
	m.updateActivityIDs = append(m.updateActivityIDs, loopID)
	return 0, 0
}

func (m *mockPushSender) EndLoopActivity(_ context.Context, loopID string, _ interface{}) (int, int) {
	m.endActivityIDs = append(m.endActivityIDs, loopID)
	return 0, 0
}

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

func seedUserAndLoop(t *testing.T, database *db.DB) (userID, loopID string) {
	t.Helper()
	userID = uuid.New().String()
	now := models.NowUTC()
	database.Exec(`INSERT INTO users (id, email, name, oauth_provider, oauth_id, created_at, updated_at)
		VALUES (?, 'test@test.com', 'Test', 'dev', ?, ?, ?)`, userID, userID, now, now)

	loopID = uuid.New().String()
	inviteCode := uuid.New().String()[:8]
	loop := &models.Loop{
		ID: loopID, Name: "Test", Description: "test", Icon: "test",
		CreatorID: userID, InviteCode: inviteCode, CreatedAt: now, UpdatedAt: now,
	}
	database.CreateLoop(context.Background(), loop)
	return
}

func seedTimedOutRequest(t *testing.T, database *db.DB, loopID, userID string) string {
	t.Helper()
	id := uuid.New().String()
	now := models.NowUTC()
	pastTimeout := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	timeoutSeconds := 60
	req := &models.Request{
		ID: id, LoopID: loopID, CreatorID: userID,
		ProcessingType: "time-sensitive", ContentType: "markdown", Priority: "medium",
		Title: "Timeout Test", RequestText: "timeout test", Platform: "api", ResponseType: "boolean",
		ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
		TimeoutSeconds: &timeoutSeconds, TimeoutAt: &pastTimeout,
		Status: "pending", CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateRequest(context.Background(), req); err != nil {
		t.Fatalf("seed request: %v", err)
	}
	return id
}

func TestTimeoutWorker_ExpiresRequests(t *testing.T) {
	database := openTestDB(t)
	userID, loopID := seedUserAndLoop(t, database)
	reqID := seedTimedOutRequest(t, database, loopID, userID)

	broker := events.NewBroker()
	mockPush := &mockPushSender{}
	worker := NewTimeoutWorker(database, mockPush, broker)

	// Run one tick manually
	worker.tick()

	// Verify request is now timed out in DB
	req, err := database.GetRequest(context.Background(), reqID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if req.Status != "timeout" {
		t.Errorf("status = %q, want %q", req.Status, "timeout")
	}
}

func TestTimeoutWorker_PublishesSSE(t *testing.T) {
	database := openTestDB(t)
	userID, loopID := seedUserAndLoop(t, database)
	reqID := seedTimedOutRequest(t, database, loopID, userID)

	broker := events.NewBroker()
	eventCh, unsub := broker.Subscribe(reqID)
	defer unsub()

	worker := NewTimeoutWorker(database, nil, broker)
	worker.tick()

	select {
	case event := <-eventCh:
		if event.Status != "timeout" {
			t.Errorf("event status = %q, want %q", event.Status, "timeout")
		}
		if event.RequestID != reqID {
			t.Errorf("event request_id = %q, want %q", event.RequestID, reqID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broker event")
	}
}

func TestTimeoutWorker_SendsPush(t *testing.T) {
	database := openTestDB(t)
	userID, loopID := seedUserAndLoop(t, database)
	seedTimedOutRequest(t, database, loopID, userID)

	mockPush := &mockPushSender{}
	worker := NewTimeoutWorker(database, mockPush, nil)
	worker.tick()

	if len(mockPush.calls) != 1 {
		t.Fatalf("push calls = %d, want 1", len(mockPush.calls))
	}
	if mockPush.calls[0].LoopID != loopID {
		t.Errorf("push loop_id = %q, want %q", mockPush.calls[0].LoopID, loopID)
	}
}

func TestTimeoutWorker_SendsEndLiveActivity(t *testing.T) {
	database := openTestDB(t)
	userID, loopID := seedUserAndLoop(t, database)
	seedTimedOutRequest(t, database, loopID, userID)

	mockPush := &mockPushSender{}
	worker := NewTimeoutWorker(database, mockPush, nil)
	worker.tick()

	// After timeout, no pending requests remain → should call EndLoopActivity
	if len(mockPush.endActivityIDs) != 1 {
		t.Fatalf("end activity calls = %d, want 1", len(mockPush.endActivityIDs))
	}
	if mockPush.endActivityIDs[0] != loopID {
		t.Errorf("end activity loop_id = %q, want %q", mockPush.endActivityIDs[0], loopID)
	}
}

func TestTimeoutWorker_SendsUpdateLiveActivity(t *testing.T) {
	database := openTestDB(t)
	userID, loopID := seedUserAndLoop(t, database)

	// Create one timed-out request AND one still-pending request
	seedTimedOutRequest(t, database, loopID, userID)

	// Create a request that is NOT timed out (still pending)
	now := models.NowUTC()
	futureTimeout := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	timeoutSeconds := 3600
	pendingReq := &models.Request{
		ID: uuid.New().String(), LoopID: loopID, CreatorID: userID,
		ProcessingType: "time-sensitive", ContentType: "markdown", Priority: "medium",
		Title: "Still Pending", RequestText: "still pending", Platform: "api", ResponseType: "boolean",
		ResponseConfig: json.RawMessage(`{"true_label":"Y","false_label":"N"}`),
		TimeoutSeconds: &timeoutSeconds, TimeoutAt: &futureTimeout,
		Status: "pending", CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateRequest(context.Background(), pendingReq); err != nil {
		t.Fatalf("seed pending request: %v", err)
	}

	mockPush := &mockPushSender{}
	worker := NewTimeoutWorker(database, mockPush, nil)
	worker.tick()

	// Still have pending requests → should call UpdateLoopActivity, not End
	if len(mockPush.updateActivityIDs) != 1 {
		t.Fatalf("update activity calls = %d, want 1", len(mockPush.updateActivityIDs))
	}
	if mockPush.updateActivityIDs[0] != loopID {
		t.Errorf("update activity loop_id = %q, want %q", mockPush.updateActivityIDs[0], loopID)
	}
	if len(mockPush.endActivityIDs) != 0 {
		t.Errorf("end activity calls = %d, want 0", len(mockPush.endActivityIDs))
	}
}

func TestTimeoutWorker_MultipleLoops(t *testing.T) {
	database := openTestDB(t)
	userID, loopID1 := seedUserAndLoop(t, database)

	// Create a second loop
	loopID2 := uuid.New().String()
	inviteCode := uuid.New().String()[:8]
	now := models.NowUTC()
	loop2 := &models.Loop{
		ID: loopID2, Name: "Test2", Description: "test2", Icon: "test",
		CreatorID: userID, InviteCode: inviteCode, CreatedAt: now, UpdatedAt: now,
	}
	database.CreateLoop(context.Background(), loop2)

	seedTimedOutRequest(t, database, loopID1, userID)
	seedTimedOutRequest(t, database, loopID2, userID)

	mockPush := &mockPushSender{}
	worker := NewTimeoutWorker(database, mockPush, nil)
	worker.tick()

	// Both loops should get end activity calls (no remaining pending)
	if len(mockPush.endActivityIDs) != 2 {
		t.Fatalf("end activity calls = %d, want 2", len(mockPush.endActivityIDs))
	}
	// Check both loop IDs are present
	loopsSeen := map[string]bool{}
	for _, id := range mockPush.endActivityIDs {
		loopsSeen[id] = true
	}
	if !loopsSeen[loopID1] {
		t.Errorf("missing end activity for loop1")
	}
	if !loopsSeen[loopID2] {
		t.Errorf("missing end activity for loop2")
	}
}

func TestTimeoutWorker_NilPushAndBroker(t *testing.T) {
	database := openTestDB(t)
	userID, loopID := seedUserAndLoop(t, database)
	seedTimedOutRequest(t, database, loopID, userID)

	// Should not panic with nil push and nil broker
	worker := NewTimeoutWorker(database, nil, nil)
	worker.tick() // no panic = pass
}

func TestTimeoutWorker_StopGraceful(t *testing.T) {
	database := openTestDB(t)
	worker := NewTimeoutWorker(database, nil, nil)
	worker.Start(100 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		worker.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return within 2 seconds")
	}
}

func TestTimeoutWorker_PanicRecovery(t *testing.T) {
	database := openTestDB(t)
	// Create a worker and close the DB to force a panic/error in tick
	database.Close()

	worker := NewTimeoutWorker(database, nil, nil)
	// tick() should recover from panic — not crash
	worker.tick()
}
