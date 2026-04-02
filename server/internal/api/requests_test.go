package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hitl-sh/handoff-server/internal/models"
)

func TestCreateRequests(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "create@test.dev", "Creator")
	apiKey := te.createAPIKey(t, jwt)
	loopID, _ := te.createLoop(t, apiKey, "Create Loop")

	tests := []struct {
		name       string
		respType   string
		respConfig map[string]interface{}
	}{
		{"single_select", "single_select", map[string]interface{}{
			"options": []map[string]string{
				{"value": "approve", "label": "Approve"},
				{"value": "reject", "label": "Reject"},
			},
		}},
		{"multi_select", "multi_select", map[string]interface{}{
			"options": []map[string]string{
				{"value": "spam", "label": "Spam"},
				{"value": "violence", "label": "Violence"},
				{"value": "ok", "label": "OK"},
			},
			"min_selections": 1, "max_selections": 2,
		}},
		{"text", "text", map[string]interface{}{
			"min_length": 5, "max_length": 1000,
		}},
		{"rating", "rating", map[string]interface{}{
			"scale_min": 1, "scale_max": 5, "scale_step": 1,
		}},
		{"number", "number", map[string]interface{}{
			"min_value": 0, "max_value": 100, "decimal_places": 2,
		}},
		{"boolean", "boolean", map[string]interface{}{
			"true_label": "Yes", "false_label": "No",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqID := te.createRequest(t, apiKey, loopID, tt.respType, tt.respConfig)
			if reqID == "" {
				t.Fatal("expected request_id")
			}
		})
	}
}

func TestListRequests(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "list@test.dev", "Lister")
	apiKey := te.createAPIKey(t, jwt)
	loopID, _ := te.createLoop(t, apiKey, "List Loop")

	// Create a few requests
	for i := 0; i < 3; i++ {
		te.createRequest(t, apiKey, loopID, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})
	}

	t.Run("list all", func(t *testing.T) {
		status, env := te.do(t, "GET", "/api/v1/requests", nil, apiKey)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data map[string]json.RawMessage
		json.Unmarshal(env.Data, &data)
		var total int
		json.Unmarshal(data["total"], &total)
		if total < 3 {
			t.Errorf("expected >= 3, got %d", total)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		status, _ := te.do(t, "GET", "/api/v1/requests?status=pending", nil, apiKey)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("filter by loop_id", func(t *testing.T) {
		status, _ := te.do(t, "GET", "/api/v1/requests?loop_id="+loopID, nil, apiKey)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		status, env := te.do(t, "GET", "/api/v1/requests?limit=1&offset=0", nil, apiKey)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data map[string]json.RawMessage
		json.Unmarshal(env.Data, &data)
		var hasMore bool
		json.Unmarshal(data["has_more"], &hasMore)
		if !hasMore {
			t.Error("expected has_more=true with limit=1")
		}
	})
}

func TestGetRequest(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "get@test.dev", "Getter")
	apiKey := te.createAPIKey(t, jwt)
	loopID, _ := te.createLoop(t, apiKey, "Get Loop")
	reqID := te.createRequest(t, apiKey, loopID, "boolean", map[string]interface{}{
		"true_label": "Y", "false_label": "N",
	})

	t.Run("get existing request", func(t *testing.T) {
		status, _ := te.do(t, "GET", "/api/v1/requests/"+reqID, nil, apiKey)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("get nonexistent request", func(t *testing.T) {
		status, _ := te.do(t, "GET", "/api/v1/requests/nonexistent-id", nil, apiKey)
		if status != 404 {
			t.Errorf("expected 404, got %d", status)
		}
	})
}

func TestCancelRequest(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "cancel@test.dev", "Canceller")
	apiKey := te.createAPIKey(t, jwt)
	loopID, inviteCode := te.createLoop(t, apiKey, "Cancel Loop")
	jwt2 := te.devLogin(t, "cancel-rev@test.dev", "Reviewer")
	te.do(t, "POST", "/api/v1/loops/join", map[string]string{"invite_code": inviteCode}, jwt2)

	t.Run("cancel pending request", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})
		status, _ := te.do(t, "DELETE", "/api/v1/requests/"+reqID, nil, apiKey)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("cannot cancel completed request", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})
		boolTrue := true
		te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{"boolean": boolTrue, "boolean_label": "Y"},
		}, jwt2)

		status, _ := te.do(t, "DELETE", "/api/v1/requests/"+reqID, nil, apiKey)
		if status != 400 {
			t.Errorf("expected 400, got %d", status)
		}
	})
}

func TestExpireTimedOutRequests(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "expire@test.dev", "Expirer")
	apiKey := te.createAPIKey(t, jwt)
	loopID, _ := te.createLoop(t, apiKey, "Timeout Loop")

	// Create a request with a 1-second timeout
	status, env := te.do(t, "POST", fmt.Sprintf("/api/v1/loops/%s/requests", loopID), map[string]interface{}{
		"processing_type": "time-sensitive",
		"type":            "markdown",
		"title":           "Timeout Request",
		"request_text":    "will timeout",
		"response_type":   "boolean",
		"response_config": map[string]interface{}{"true_label": "Y", "false_label": "N"},
		"timeout_seconds": 60,
	}, apiKey)
	if status != 201 {
		t.Fatalf("create request failed: %d %s", status, env.Msg)
	}
	reqID := dataField(t, env, "request_id")

	// Manually set timeout_at to the past so it's already expired
	now := models.NowUTC()
	_, err := te.db.Exec("UPDATE requests SET timeout_at = datetime(?, '-10 seconds') WHERE id = ?", now, reqID)
	if err != nil {
		t.Fatalf("update timeout_at: %v", err)
	}

	// Run the expire method
	expired, err := te.db.ExpireTimedOutRequests(context.Background())
	if err != nil {
		t.Fatalf("ExpireTimedOutRequests: %v", err)
	}

	if len(expired) != 1 {
		t.Fatalf("expected 1 expired request, got %d", len(expired))
	}
	if expired[0].ID != reqID {
		t.Errorf("expected expired request ID %s, got %s", reqID, expired[0].ID)
	}

	// Verify the request status is now 'timeout' via API
	status, env = te.do(t, "GET", "/api/v1/requests/"+reqID, nil, apiKey)
	if status != 200 {
		t.Fatalf("get request failed: %d", status)
	}
	var reqData map[string]interface{}
	json.Unmarshal(env.Data, &reqData)
	req := reqData["request"].(map[string]interface{})
	if req["status"] != "timeout" {
		t.Errorf("expected status 'timeout', got %q", req["status"])
	}
}
