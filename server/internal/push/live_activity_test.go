package push

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hitl-sh/handoff-server/internal/models"
)

func TestBuildLiveActivityPayload_WithAlert(t *testing.T) {
	cs := map[string]interface{}{"pendingRequests": []interface{}{}, "totalPending": 1}
	stale := int64(1700000000)

	raw, err := BuildLiveActivityPayload(cs, &stale, "New request", "Deploy v2?")
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	aps, ok := m["aps"].(map[string]interface{})
	if !ok {
		t.Fatal("missing aps")
	}

	if aps["event"] != "update" {
		t.Errorf("event = %v, want %q", aps["event"], "update")
	}
	if aps["timestamp"] == nil {
		t.Error("timestamp should not be nil")
	}
	if aps["stale-date"] == nil {
		t.Error("stale-date should not be nil")
	}
	if aps["sound"] != "default" {
		t.Errorf("sound = %v, want %q", aps["sound"], "default")
	}

	alert, ok := aps["alert"].(map[string]interface{})
	if !ok {
		t.Fatal("missing alert")
	}
	if alert["title"] != "New request" {
		t.Errorf("alert.title = %v, want %q", alert["title"], "New request")
	}
	if alert["body"] != "Deploy v2?" {
		t.Errorf("alert.body = %v, want %q", alert["body"], "Deploy v2?")
	}
}

func TestBuildLiveActivityPayload_NoAlert(t *testing.T) {
	cs := map[string]interface{}{"totalPending": 0}

	raw, err := BuildLiveActivityPayload(cs, nil, "", "")
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(raw, &m)
	aps := m["aps"].(map[string]interface{})

	if _, ok := aps["alert"]; ok {
		t.Error("alert should not be present when title is empty")
	}
	if _, ok := aps["sound"]; ok {
		t.Error("sound should not be present when no alert")
	}
	if _, ok := aps["stale-date"]; ok {
		t.Error("stale-date should not be present when nil")
	}
}

func TestBuildLiveActivityEndPayload(t *testing.T) {
	cs := map[string]interface{}{"pendingRequests": []interface{}{}, "totalPending": 0}
	dismissal := time.Now().Add(4 * time.Hour).Unix()

	raw, err := BuildLiveActivityEndPayload(cs, dismissal)
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(raw, &m)
	aps := m["aps"].(map[string]interface{})

	if aps["event"] != "end" {
		t.Errorf("event = %v, want %q", aps["event"], "end")
	}
	if aps["dismissal-date"] == nil {
		t.Error("dismissal-date should not be nil")
	}
	if aps["content-state"] == nil {
		t.Error("content-state should not be nil")
	}
}

func TestBuildLoopContentState_Empty(t *testing.T) {
	cs := BuildLoopContentState(nil, 0)

	if cs["totalPending"] != 0 {
		t.Errorf("totalPending = %v, want 0", cs["totalPending"])
	}
	items := cs["pendingRequests"].([]map[string]interface{})
	if len(items) != 0 {
		t.Errorf("pendingRequests = %d, want 0", len(items))
	}
}

func TestBuildLoopContentState_WithRequests(t *testing.T) {
	timeout := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	reqs := []models.Request{
		{
			ID:           "req-1",
			Title:        "Deploy Request",
			RequestText:  "Deploy to production?",
			ResponseType: "boolean",
			Priority:     "critical",
			TimeoutAt:    &timeout,
		},
		{
			ID:           "req-2",
			Title:        "Budget Approval",
			RequestText:  "Approve budget increase",
			ResponseType: "text",
			Priority:     "medium",
			TimeoutAt:    nil,
		},
	}

	cs := BuildLoopContentState(reqs, 5)

	if cs["totalPending"] != 5 {
		t.Errorf("totalPending = %v, want 5", cs["totalPending"])
	}

	items := cs["pendingRequests"].([]map[string]interface{})
	if len(items) != 2 {
		t.Fatalf("pendingRequests = %d, want 2", len(items))
	}

	// First item
	if items[0]["id"] != "req-1" {
		t.Errorf("items[0].id = %v, want %q", items[0]["id"], "req-1")
	}
	if items[0]["priority"] != "critical" {
		t.Errorf("items[0].priority = %v, want %q", items[0]["priority"], "critical")
	}
	if items[0]["responseType"] != "boolean" {
		t.Errorf("items[0].responseType = %v, want %q", items[0]["responseType"], "boolean")
	}
	if items[0]["timeoutAt"] == nil {
		t.Error("items[0].timeoutAt should not be nil (has timeout)")
	}

	// Second item
	if items[1]["timeoutAt"] != nil {
		t.Errorf("items[1].timeoutAt = %v, want nil", items[1]["timeoutAt"])
	}
}

func TestBuildLoopContentState_TruncatesText(t *testing.T) {
	longText := ""
	for i := 0; i < 100; i++ {
		longText += "a"
	}
	reqs := []models.Request{
		{
			ID:           "req-1",
			Title:        "Long Text Request",
			RequestText:  longText,
			ResponseType: "boolean",
			Priority:     "medium",
		},
	}

	cs := BuildLoopContentState(reqs, 1)
	items := cs["pendingRequests"].([]map[string]interface{})
	text := items[0]["text"].(string)
	runes := []rune(text)
	if len(runes) > 60 {
		t.Errorf("text length = %d runes, want <= 60", len(runes))
	}
}

func TestBuildLoopContentState_TimeoutAtAppleEpoch(t *testing.T) {
	// Verify the Apple reference epoch offset is applied
	timeout := "2024-06-15T12:00:00Z"
	reqs := []models.Request{
		{
			ID:           "req-1",
			Title:        "Test Request",
			RequestText:  "test",
			ResponseType: "boolean",
			Priority:     "medium",
			TimeoutAt:    &timeout,
		},
	}

	cs := BuildLoopContentState(reqs, 1)
	items := cs["pendingRequests"].([]map[string]interface{})
	timeoutAt := items[0]["timeoutAt"].(float64)

	// Parse the timeout and check the offset
	parsed, _ := time.Parse(time.RFC3339, timeout)
	expected := float64(parsed.Unix()) - float64(AppleReferenceEpochOffset)
	if timeoutAt != expected {
		t.Errorf("timeoutAt = %v, want %v (Unix %d - Apple epoch %d)",
			timeoutAt, expected, parsed.Unix(), AppleReferenceEpochOffset)
	}
}

func TestEarliestTimeoutUnix_NoTimeouts(t *testing.T) {
	reqs := []models.Request{
		{ID: "req-1", TimeoutAt: nil},
		{ID: "req-2", TimeoutAt: nil},
	}

	result := EarliestTimeoutUnix(reqs)
	if result != nil {
		t.Errorf("result = %v, want nil", *result)
	}
}

func TestEarliestTimeoutUnix_Empty(t *testing.T) {
	result := EarliestTimeoutUnix(nil)
	if result != nil {
		t.Errorf("result = %v, want nil", *result)
	}
}

func TestEarliestTimeoutUnix_FindsEarliest(t *testing.T) {
	early := "2024-06-15T10:00:00Z"
	late := "2024-06-15T14:00:00Z"
	reqs := []models.Request{
		{ID: "req-1", TimeoutAt: &late},
		{ID: "req-2", TimeoutAt: nil},
		{ID: "req-3", TimeoutAt: &early},
	}

	result := EarliestTimeoutUnix(reqs)
	if result == nil {
		t.Fatal("result should not be nil")
	}

	parsed, _ := time.Parse(time.RFC3339, early)
	if *result != parsed.Unix() {
		t.Errorf("result = %d, want %d", *result, parsed.Unix())
	}
}

func TestEarliestTimeoutUnix_InvalidTimeFormat(t *testing.T) {
	invalid := "not-a-date"
	valid := "2024-06-15T10:00:00Z"
	reqs := []models.Request{
		{ID: "req-1", TimeoutAt: &invalid},
		{ID: "req-2", TimeoutAt: &valid},
	}

	result := EarliestTimeoutUnix(reqs)
	if result == nil {
		t.Fatal("result should not be nil (valid date exists)")
	}

	parsed, _ := time.Parse(time.RFC3339, valid)
	if *result != parsed.Unix() {
		t.Errorf("result = %d, want %d", *result, parsed.Unix())
	}
}
