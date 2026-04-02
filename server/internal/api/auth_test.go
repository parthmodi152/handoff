package api_test

import "testing"

func TestHealthEndpoint(t *testing.T) {
	te := setupTestEnv(t)
	status, env := te.do(t, "GET", "/health", nil, "")
	if status != 200 {
		t.Errorf("expected 200, got %d", status)
	}
	if env.Error {
		t.Error("expected no error")
	}
}

func TestDevLogin(t *testing.T) {
	te := setupTestEnv(t)

	t.Run("creates user and returns JWT", func(t *testing.T) {
		status, env := te.do(t, "POST", "/auth/dev-login", map[string]string{
			"email": "test@handoff.dev", "name": "Test User",
		}, "")
		if status != 200 {
			t.Fatalf("expected 200, got %d: %s", status, env.Msg)
		}
		token := dataField(t, env, "token")
		if token == "" {
			t.Fatal("expected token")
		}
	})

	t.Run("defaults email and name", func(t *testing.T) {
		status, env := te.do(t, "POST", "/auth/dev-login", map[string]string{}, "")
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		if dataField(t, env, "user", "email") != "dev@handoff.local" {
			t.Error("expected default email")
		}
	})

	t.Run("reuses existing user", func(t *testing.T) {
		_, env1 := te.do(t, "POST", "/auth/dev-login", map[string]string{
			"email": "repeat@test.dev", "name": "Repeat",
		}, "")
		_, env2 := te.do(t, "POST", "/auth/dev-login", map[string]string{
			"email": "repeat@test.dev", "name": "Repeat",
		}, "")
		if dataField(t, env1, "user", "id") != dataField(t, env2, "user", "id") {
			t.Error("expected same user id on repeat login")
		}
	})
}

func TestAuthMe(t *testing.T) {
	te := setupTestEnv(t)

	t.Run("returns user for valid JWT", func(t *testing.T) {
		jwt := te.devLogin(t, "me@test.dev", "Me")
		status, env := te.do(t, "GET", "/auth/me", nil, jwt)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		if dataField(t, env, "user", "email") != "me@test.dev" {
			t.Error("wrong email")
		}
	})

	t.Run("rejects missing auth", func(t *testing.T) {
		status, _ := te.do(t, "GET", "/auth/me", nil, "")
		if status != 401 {
			t.Errorf("expected 401, got %d", status)
		}
	})

	t.Run("rejects invalid JWT", func(t *testing.T) {
		status, _ := te.do(t, "GET", "/auth/me", nil, "eyJinvalid.token.here")
		if status != 401 {
			t.Errorf("expected 401, got %d", status)
		}
	})
}
