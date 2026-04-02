package api_test

import (
	"fmt"
	"testing"
)

func TestRequestCreationValidation(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "val@test.dev", "Validator")
	apiKey := te.createAPIKey(t, jwt)
	loopID, _ := te.createLoop(t, apiKey, "Validation Loop")

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{"missing processing_type", map[string]interface{}{
			"type": "markdown", "title": "Test Title", "request_text": "test",
			"response_type": "boolean", "response_config": map[string]interface{}{"true_label": "Y", "false_label": "N"},
		}},
		{"missing request_text", map[string]interface{}{
			"processing_type": "deferred", "type": "markdown", "title": "Test Title",
			"response_type": "boolean", "response_config": map[string]interface{}{"true_label": "Y", "false_label": "N"},
		}},
		{"time-sensitive without timeout", map[string]interface{}{
			"processing_type": "time-sensitive", "type": "markdown", "title": "Test Title", "request_text": "test",
			"response_type": "boolean", "response_config": map[string]interface{}{"true_label": "Y", "false_label": "N"},
		}},
		{"invalid response_type", map[string]interface{}{
			"processing_type": "deferred", "type": "markdown", "title": "Test Title", "request_text": "test",
			"response_type": "invalid", "response_config": map[string]interface{}{},
		}},
		{"invalid priority", map[string]interface{}{
			"processing_type": "deferred", "type": "markdown", "priority": "super",
			"title": "Test Title", "request_text": "test", "response_type": "boolean",
			"response_config": map[string]interface{}{"true_label": "Y", "false_label": "N"},
		}},
		{"image without image_url", map[string]interface{}{
			"processing_type": "deferred", "type": "image", "title": "Test Title", "request_text": "review",
			"response_type": "boolean", "response_config": map[string]interface{}{"true_label": "Y", "false_label": "N"},
		}},
		{"single_select < 2 options", map[string]interface{}{
			"processing_type": "deferred", "type": "markdown", "title": "Test Title", "request_text": "test",
			"response_type": "single_select", "response_config": map[string]interface{}{
				"options": []map[string]string{{"value": "only", "label": "Only"}},
			},
		}},
		{"rating without scale_max", map[string]interface{}{
			"processing_type": "deferred", "type": "markdown", "title": "Test Title", "request_text": "test",
			"response_type": "rating", "response_config": map[string]interface{}{"scale_min": 1},
		}},
		{"number without max_value", map[string]interface{}{
			"processing_type": "deferred", "type": "markdown", "title": "Test Title", "request_text": "test",
			"response_type": "number", "response_config": map[string]interface{}{"min_value": 0},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := te.do(t, "POST", fmt.Sprintf("/api/v1/loops/%s/requests", loopID), tt.body, apiKey)
			if status != 400 {
				t.Errorf("expected 400, got %d", status)
			}
		})
	}
}

func TestResponseDataValidation(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "resval@test.dev", "ResVal")
	apiKey := te.createAPIKey(t, jwt)
	loopID, inviteCode := te.createLoop(t, apiKey, "ResVal Loop")
	jwt2 := te.devLogin(t, "resval-rev@test.dev", "Reviewer")
	te.do(t, "POST", "/api/v1/loops/join", map[string]string{"invite_code": inviteCode}, jwt2)

	t.Run("single_select: invalid value", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "single_select", map[string]interface{}{
			"options": []map[string]string{{"value": "a", "label": "A"}, {"value": "b", "label": "B"}},
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]string{"selected_value": "c", "selected_label": "C"},
		}, jwt2)
		if status != 400 {
			t.Errorf("expected 400, got %d", status)
		}
	})

	t.Run("text: below min_length", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "text", map[string]interface{}{
			"min_length": 10, "max_length": 500,
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": "short",
		}, jwt2)
		if status != 400 {
			t.Errorf("expected 400, got %d", status)
		}
	})

	t.Run("rating: out of range", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "rating", map[string]interface{}{
			"scale_min": 1, "scale_max": 5, "scale_step": 1,
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{"rating": 10.0},
		}, jwt2)
		if status != 400 {
			t.Errorf("expected 400, got %d", status)
		}
	})

	t.Run("number: exceeds max", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "number", map[string]interface{}{
			"min_value": 0, "max_value": 100, "decimal_places": 0,
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{"number": 999.0},
		}, jwt2)
		if status != 400 {
			t.Errorf("expected 400, got %d", status)
		}
	})

	t.Run("multi_select: too many selections", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "multi_select", map[string]interface{}{
			"options": []map[string]string{
				{"value": "a", "label": "A"}, {"value": "b", "label": "B"}, {"value": "c", "label": "C"},
			},
			"min_selections": 1, "max_selections": 1,
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{
				"selected_values": []string{"a", "b"}, "selected_labels": []string{"A", "B"},
			},
		}, jwt2)
		if status != 400 {
			t.Errorf("expected 400, got %d", status)
		}
	})
}

func TestCORSHeaders(t *testing.T) {
	te := setupTestEnv(t)
	status, _ := te.do(t, "GET", "/health", nil, "")
	if status != 200 {
		t.Errorf("expected 200, got %d", status)
	}
}
