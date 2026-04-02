package api_test

import (
	"encoding/json"
	"testing"
)

func TestLoops(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "loops@test.dev", "Loop User")
	apiKey := te.createAPIKey(t, jwt)

	t.Run("create loop with QR", func(t *testing.T) {
		status, env := te.do(t, "POST", "/api/v1/loops", map[string]interface{}{
			"name": "Content Review", "description": "Review content", "icon": "shield-check",
		}, apiKey)
		if status != 201 {
			t.Fatalf("expected 201, got %d: %s", status, env.Msg)
		}
		if id := dataField(t, env, "loop", "id"); id == "" {
			t.Error("missing loop id")
		}
		code := dataField(t, env, "loop", "invite_code")
		if len(code) != 6 {
			t.Errorf("expected 6-char code, got %q", code)
		}
		qr := dataField(t, env, "loop", "invite_qr")
		if len(qr) < 10 {
			t.Error("expected QR base64")
		}
	})

	t.Run("list loops", func(t *testing.T) {
		status, env := te.do(t, "GET", "/api/v1/loops", nil, apiKey)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data map[string]json.RawMessage
		json.Unmarshal(env.Data, &data)
		var count int
		json.Unmarshal(data["count"], &count)
		if count < 1 {
			t.Errorf("expected >= 1, got %d", count)
		}
	})

	t.Run("get loop with members", func(t *testing.T) {
		loopID, _ := te.createLoop(t, apiKey, "Get Test")
		status, _ := te.do(t, "GET", "/api/v1/loops/"+loopID, nil, apiKey)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("reject empty name", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/loops", map[string]interface{}{"name": ""}, apiKey)
		if status != 400 {
			t.Errorf("expected 400, got %d", status)
		}
	})

	t.Run("non-member cannot access", func(t *testing.T) {
		loopID, _ := te.createLoop(t, apiKey, "Private")
		jwt2 := te.devLogin(t, "outsider@test.dev", "Outsider")
		key2 := te.createAPIKey(t, jwt2)
		status, _ := te.do(t, "GET", "/api/v1/loops/"+loopID, nil, key2)
		if status != 403 {
			t.Errorf("expected 403, got %d", status)
		}
	})
}

func TestJoinLoop(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "owner@test.dev", "Owner")
	apiKey := te.createAPIKey(t, jwt)

	t.Run("join by invite code", func(t *testing.T) {
		_, code := te.createLoop(t, apiKey, "Joinable")
		jwt2 := te.devLogin(t, "joiner@test.dev", "Joiner")
		status, _ := te.do(t, "POST", "/api/v1/loops/join", map[string]string{"invite_code": code}, jwt2)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("cannot join twice", func(t *testing.T) {
		_, code := te.createLoop(t, apiKey, "Double")
		jwt2 := te.devLogin(t, "double@test.dev", "Double")
		te.do(t, "POST", "/api/v1/loops/join", map[string]string{"invite_code": code}, jwt2)
		status, _ := te.do(t, "POST", "/api/v1/loops/join", map[string]string{"invite_code": code}, jwt2)
		if status != 409 {
			t.Errorf("expected 409, got %d", status)
		}
	})

	t.Run("invalid code returns 404", func(t *testing.T) {
		jwt2 := te.devLogin(t, "badcode@test.dev", "Bad")
		status, _ := te.do(t, "POST", "/api/v1/loops/join", map[string]string{"invite_code": "ZZZZZZ"}, jwt2)
		if status != 404 {
			t.Errorf("expected 404, got %d", status)
		}
	})
}
