package api_test

import (
	"encoding/json"
	"testing"
)

func TestRespondToRequest(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "respond@test.dev", "Requester")
	apiKey := te.createAPIKey(t, jwt)
	loopID, inviteCode := te.createLoop(t, apiKey, "Respond Loop")
	jwt2 := te.devLogin(t, "respond-rev@test.dev", "Reviewer")
	te.do(t, "POST", "/api/v1/loops/join", map[string]string{"invite_code": inviteCode}, jwt2)

	t.Run("single_select", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "single_select", map[string]interface{}{
			"options": []map[string]string{
				{"value": "approve", "label": "Approve"},
				{"value": "reject", "label": "Reject"},
			},
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]string{
				"selected_value": "approve", "selected_label": "Approve",
			},
		}, jwt2)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("boolean", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "boolean", map[string]interface{}{
			"true_label": "Yes", "false_label": "No",
		})
		boolTrue := true
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{"boolean": boolTrue, "boolean_label": "Yes"},
		}, jwt2)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("text", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "text", map[string]interface{}{
			"min_length": 5, "max_length": 500,
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": "This is my detailed feedback on the content.",
		}, jwt2)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("rating", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "rating", map[string]interface{}{
			"scale_min": 1, "scale_max": 5, "scale_step": 1,
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{"rating": 4.0},
		}, jwt2)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("number", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "number", map[string]interface{}{
			"min_value": 0, "max_value": 1000, "decimal_places": 2,
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{"number": 299.99},
		}, jwt2)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("multi_select", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "multi_select", map[string]interface{}{
			"options": []map[string]string{
				{"value": "spam", "label": "Spam"},
				{"value": "violence", "label": "Violence"},
				{"value": "ok", "label": "OK"},
			},
			"min_selections": 1, "max_selections": 2,
		})
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{
				"selected_values": []string{"spam", "violence"},
				"selected_labels": []string{"Spam", "Violence"},
			},
		}, jwt2)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("status changes to completed", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})
		boolFalse := false
		te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{"boolean": boolFalse, "boolean_label": "N"},
		}, jwt2)

		_, env := te.do(t, "GET", "/api/v1/requests/"+reqID, nil, apiKey)
		var data map[string]json.RawMessage
		json.Unmarshal(env.Data, &data)
		var req map[string]json.RawMessage
		json.Unmarshal(data["request"], &req)
		var status string
		json.Unmarshal(req["status"], &status)
		if status != "completed" {
			t.Errorf("expected completed, got %s", status)
		}
	})

	t.Run("cannot respond twice", func(t *testing.T) {
		reqID := te.createRequest(t, apiKey, loopID, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})
		boolTrue := true
		te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{"boolean": boolTrue, "boolean_label": "Y"},
		}, jwt2)
		status, _ := te.do(t, "POST", "/api/v1/requests/"+reqID+"/respond", map[string]interface{}{
			"response_data": map[string]interface{}{"boolean": boolTrue, "boolean_label": "Y"},
		}, jwt2)
		if status != 400 {
			t.Errorf("expected 400, got %d", status)
		}
	})
}
