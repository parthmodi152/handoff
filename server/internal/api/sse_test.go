package api_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hitl-sh/handoff-server/internal/api"
	"github.com/hitl-sh/handoff-server/internal/config"
	"github.com/hitl-sh/handoff-server/internal/events"

	"net/http/httptest"
)

// setupTestEnvWithBroker creates a test environment with a real event broker.
func setupTestEnvWithBroker(t *testing.T) (*testEnv, *events.Broker) {
	t.Helper()

	te := setupTestEnvBase(t)
	broker := events.NewBroker()

	cfg := &config.Config{
		Port:      "0",
		DBPath:    te.cfg.DBPath,
		BaseURL:   "http://localhost",
		JWTSecret: "test-secret-key-for-testing-32b!",
		DevMode:   true,
	}

	router := api.NewRouter(te.db, cfg, nil, broker, nil)
	server := httptest.NewServer(router)

	t.Cleanup(func() {
		server.Close()
	})

	te.server = server
	te.cfg = cfg
	return te, broker
}

func TestSSEAlreadyCompletedRequest(t *testing.T) {
	te, _ := setupTestEnvWithBroker(t)

	// Create users and request
	apiKey := te.createAPIKey(t, te.devLogin(t, "sse-creator@test.dev", "Creator"))
	loopID, inviteCode := te.createLoop(t, apiKey, "SSE Loop")
	reviewer := te.devLogin(t, "sse-reviewer@test.dev", "Reviewer")
	te.do(t, "POST", "/api/v1/loops/join", map[string]string{"invite_code": inviteCode}, reviewer)

	reqID := te.createRequest(t, apiKey, loopID, "boolean", map[string]interface{}{
		"true_label": "Y", "false_label": "N",
	})

	// Respond to it first
	te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
		"response_data": map[string]interface{}{"boolean": true, "boolean_label": "Y"},
	}, reviewer)

	// Now connect SSE — should get event immediately
	req, _ := http.NewRequest("GET", te.server.URL+"/api/v1/requests/"+reqID+"/events", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SSE request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", ct)
	}

	// Read SSE event
	scanner := bufio.NewScanner(resp.Body)
	var eventLine, dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			eventLine = line
		}
		if strings.HasPrefix(line, "data:") {
			dataLine = line
			break
		}
	}

	if !strings.Contains(eventLine, "completed") {
		t.Errorf("expected event type 'completed', got %q", eventLine)
	}
	if dataLine == "" {
		t.Fatal("no data line received")
	}

	// Parse data
	jsonData := strings.TrimPrefix(dataLine, "data: ")
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		t.Fatalf("failed to parse SSE data: %v", err)
	}
	if data["status"] != "completed" {
		t.Errorf("expected status completed, got %v", data["status"])
	}
}

func TestSSEPendingThenRespond(t *testing.T) {
	te, broker := setupTestEnvWithBroker(t)

	apiKey := te.createAPIKey(t, te.devLogin(t, "sse2-creator@test.dev", "Creator"))
	loopID, inviteCode := te.createLoop(t, apiKey, "SSE Loop 2")
	reviewer := te.devLogin(t, "sse2-reviewer@test.dev", "Reviewer")
	te.do(t, "POST", "/api/v1/loops/join", map[string]string{"invite_code": inviteCode}, reviewer)

	reqID := te.createRequest(t, apiKey, loopID, "boolean", map[string]interface{}{
		"true_label": "Y", "false_label": "N",
	})

	// Connect SSE in background
	resultCh := make(chan string, 1)
	go func() {
		req, _ := http.NewRequest("GET", te.server.URL+"/api/v1/requests/"+reqID+"/events", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			resultCh <- fmt.Sprintf("error: %v", err)
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				resultCh <- strings.TrimPrefix(line, "data: ")
				return
			}
		}
		resultCh <- "error: no data received"
	}()

	// Give SSE connection time to establish
	time.Sleep(100 * time.Millisecond)

	// Publish event via broker (simulating what respond handler does)
	eventData, _ := json.Marshal(map[string]interface{}{
		"request": map[string]interface{}{
			"id":     reqID,
			"status": "completed",
		},
	})
	broker.Publish(events.Event{
		RequestID: reqID,
		LoopID:    loopID,
		Status:    "completed",
		Data:      eventData,
	})

	select {
	case result := <-resultCh:
		if strings.HasPrefix(result, "error:") {
			t.Fatal(result)
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result), &data); err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		req := data["request"].(map[string]interface{})
		if req["status"] != "completed" {
			t.Errorf("expected completed, got %v", req["status"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
}

func TestSSENotFound(t *testing.T) {
	te, _ := setupTestEnvWithBroker(t)
	apiKey := te.createAPIKey(t, te.devLogin(t, "sse-nf@test.dev", "NotFound"))

	status, _ := te.do(t, "GET", "/api/v1/requests/nonexistent-id/events", nil, apiKey)
	if status != 404 {
		t.Errorf("expected 404, got %d", status)
	}
}

func TestSSENoAuth(t *testing.T) {
	te, _ := setupTestEnvWithBroker(t)

	status, _ := te.do(t, "GET", "/api/v1/requests/some-id/events", nil, "")
	if status != 401 {
		t.Errorf("expected 401, got %d", status)
	}
}
