package api_test

import (
	"net/http"
	"testing"
)

func TestActivityTokenRegistration(t *testing.T) {
	te := setupTestEnv(t)
	token := te.devLogin(t, "activity-test@dev.com", "Activity Tester")
	loopID, _ := te.createLoop(t, token, "Activity Loop")

	t.Run("register ios_activity token", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "activity-token-123",
			"platform": "ios_activity",
			"loop_id":  loopID,
		}, token)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
	})

	t.Run("ios_activity without loop_id fails", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "activity-token-456",
			"platform": "ios_activity",
		}, token)
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", status)
		}
	})

	t.Run("ios_activity with empty loop_id fails", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "activity-token-789",
			"platform": "ios_activity",
			"loop_id":  "",
		}, token)
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", status)
		}
	})

	t.Run("upsert same activity token", func(t *testing.T) {
		// First register
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "upsert-activity-token",
			"platform": "ios_activity",
			"loop_id":  loopID,
		}, token)
		if status != http.StatusOK {
			t.Fatalf("first register expected 200, got %d", status)
		}

		// Second register with same token (upsert)
		status, _ = te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "upsert-activity-token",
			"platform": "ios_activity",
			"loop_id":  loopID,
		}, token)
		if status != http.StatusOK {
			t.Fatalf("upsert expected 200, got %d", status)
		}
	})

	t.Run("unregister activity token", func(t *testing.T) {
		// Register
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "to-delete-activity",
			"platform": "ios_activity",
			"loop_id":  loopID,
		}, token)
		if status != http.StatusOK {
			t.Fatalf("register expected 200, got %d", status)
		}

		// Unregister
		status, _ = te.do(t, "DELETE", "/api/v1/devices/to-delete-activity", nil, token)
		if status != http.StatusOK {
			t.Fatalf("unregister expected 200, got %d", status)
		}
	})

	t.Run("regular ios still works", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "regular-ios-token",
			"platform": "ios",
		}, token)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
	})

	t.Run("loop_id ignored for regular ios", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "ios-with-loop-id",
			"platform": "ios",
			"loop_id":  loopID,
		}, token)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
	})

	t.Run("requires auth", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "no-auth-activity",
			"platform": "ios_activity",
			"loop_id":  loopID,
		}, "")
		if status != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", status)
		}
	})
}
