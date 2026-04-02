package api_test

import (
	"net/http"
	"testing"
)

func TestDeviceTokenRegistration(t *testing.T) {
	te := setupTestEnv(t)
	token := te.devLogin(t, "device-test@dev.com", "Device Tester")

	t.Run("register device token", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "aabbccdd11223344",
			"platform": "ios",
		}, token)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
	})

	t.Run("register without token fails", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"platform": "ios",
		}, token)
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", status)
		}
	})

	t.Run("register with invalid platform fails", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "aabbccdd11223344",
			"platform": "android",
		}, token)
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", status)
		}
	})

	t.Run("upsert same token updates user", func(t *testing.T) {
		// Register as first user
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "shared-token-123",
			"platform": "ios",
		}, token)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		// Register same token as different user
		te2 := setupTestEnv(t)
		token2 := te2.devLogin(t, "device-test2@dev.com", "Device Tester 2")
		status, _ = te2.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "shared-token-123",
			"platform": "ios",
		}, token2)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
	})

	t.Run("unregister device token", func(t *testing.T) {
		// Register first
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "to-be-removed",
			"platform": "ios",
		}, token)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}

		// Unregister
		status, _ = te.do(t, "DELETE", "/api/v1/devices/to-be-removed", nil, token)
		if status != http.StatusOK {
			t.Fatalf("expected 200 on delete, got %d", status)
		}
	})

	t.Run("register requires auth", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "no-auth-token",
			"platform": "ios",
		}, "")
		if status != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", status)
		}
	})

	t.Run("register with api key fails (session only)", func(t *testing.T) {
		apiKey := te.createAPIKey(t, token)

		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token":    "api-key-token",
			"platform": "ios",
		}, apiKey)
		if status != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", status)
		}
	})

	t.Run("defaults platform to ios", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/devices", map[string]interface{}{
			"token": "default-platform-token",
		}, token)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d", status)
		}
	})
}
