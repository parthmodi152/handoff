package api_test

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestAPIKeys(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "keys@test.dev", "Keys User")

	t.Run("create returns hnd_ prefixed key", func(t *testing.T) {
		status, env := te.do(t, "POST", "/api/v1/api-keys", map[string]string{"name": "My Key"}, jwt)
		if status != 201 {
			t.Fatalf("expected 201, got %d: %s", status, env.Msg)
		}
		key := dataField(t, env, "api_key", "key")
		if len(key) < 8 || key[:4] != "hnd_" {
			t.Errorf("bad key format: %s", key)
		}
	})

	t.Run("list keys", func(t *testing.T) {
		status, env := te.do(t, "GET", "/api/v1/api-keys", nil, jwt)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data map[string]json.RawMessage
		json.Unmarshal(env.Data, &data)
		var count int
		json.Unmarshal(data["count"], &count)
		if count < 1 {
			t.Errorf("expected >= 1 key, got %d", count)
		}
	})

	t.Run("API key auth works on test endpoint", func(t *testing.T) {
		apiKey := te.createAPIKey(t, jwt)
		status, _ := te.do(t, "GET", "/api/v1/test", nil, apiKey)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("revoke key then auth fails", func(t *testing.T) {
		_, env := te.do(t, "POST", "/api/v1/api-keys", map[string]string{"name": "Revokable"}, jwt)
		keyID := dataField(t, env, "api_key", "id")
		rawKey := dataField(t, env, "api_key", "key")

		status, _ := te.do(t, "DELETE", "/api/v1/api-keys/"+keyID, nil, jwt)
		if status != 200 {
			t.Fatalf("revoke failed: %d", status)
		}

		status2, _ := te.do(t, "GET", "/api/v1/test", nil, rawKey)
		if status2 != 401 {
			t.Errorf("expected 401 for revoked key, got %d", status2)
		}
	})

	t.Run("reject empty name", func(t *testing.T) {
		status, _ := te.do(t, "POST", "/api/v1/api-keys", map[string]string{"name": ""}, jwt)
		if status != 400 {
			t.Errorf("expected 400, got %d", status)
		}
	})
}

func TestAPIKeys_AgentName(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "agent-key@test.dev", "Agent Key User")

	t.Run("create with agent_name", func(t *testing.T) {
		agent := "Claude"
		_, env := te.do(t, "POST", "/api/v1/api-keys", map[string]interface{}{
			"name":       "Agent Key",
			"agent_name": agent,
		}, jwt)
		got := dataField(t, env, "api_key", "agent_name")
		if got != "Claude" {
			t.Errorf("agent_name = %q, want %q", got, "Claude")
		}
	})

	t.Run("agent_name appears in list", func(t *testing.T) {
		status, env := te.do(t, "GET", "/api/v1/api-keys", nil, jwt)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data struct {
			APIKeys []struct {
				AgentName *string `json:"agent_name"`
			} `json:"api_keys"`
		}
		json.Unmarshal(env.Data, &data)
		found := false
		for _, k := range data.APIKeys {
			if k.AgentName != nil && *k.AgentName == "Claude" {
				found = true
			}
		}
		if !found {
			t.Error("agent_name 'Claude' not found in listed keys")
		}
	})

	t.Run("empty agent_name treated as nil", func(t *testing.T) {
		status, env := te.do(t, "POST", "/api/v1/api-keys", map[string]interface{}{
			"name":       "Empty Agent",
			"agent_name": "   ",
		}, jwt)
		if status != 201 {
			t.Fatalf("expected 201, got %d: %s", status, env.Msg)
		}
		// agent_name should be absent (trimmed empty → nil → omitted)
		var data map[string]json.RawMessage
		json.Unmarshal(env.Data, &data)
		var keyData map[string]json.RawMessage
		json.Unmarshal(data["api_key"], &keyData)
		if _, exists := keyData["agent_name"]; exists {
			t.Error("expected agent_name to be omitted for empty string")
		}
	})

	t.Run("reject agent_name over 100 chars", func(t *testing.T) {
		long := string(make([]byte, 101))
		for i := range long {
			long = long[:i] + "a" + long[i+1:]
		}
		status, _ := te.do(t, "POST", "/api/v1/api-keys", map[string]interface{}{
			"name":       "Long Agent",
			"agent_name": string(make([]byte, 101)),
		}, jwt)
		if status != 400 {
			t.Errorf("expected 400 for long agent_name, got %d", status)
		}
	})
}

func TestAPIKeys_AllowedLoops(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "scoped-key@test.dev", "Scoped Key User")
	loop1, _ := te.createLoop(t, jwt, "Scoped Loop A")
	loop2, _ := te.createLoop(t, jwt, "Scoped Loop B")

	t.Run("create with allowed_loops", func(t *testing.T) {
		status, env := te.do(t, "POST", "/api/v1/api-keys", map[string]interface{}{
			"name":          "Scoped Key",
			"allowed_loops": []string{loop1},
		}, jwt)
		if status != 201 {
			t.Fatalf("expected 201, got %d: %s", status, env.Msg)
		}
		var data struct {
			APIKey struct {
				AllowedLoops []string `json:"allowed_loops"`
			} `json:"api_key"`
		}
		json.Unmarshal(env.Data, &data)
		if len(data.APIKey.AllowedLoops) != 1 || data.APIKey.AllowedLoops[0] != loop1 {
			t.Errorf("allowed_loops = %v, want [%s]", data.APIKey.AllowedLoops, loop1)
		}
	})

	t.Run("reject loop user is not member of", func(t *testing.T) {
		jwt2 := te.devLogin(t, "other@test.dev", "Other")
		otherLoop, _ := te.createLoop(t, jwt2, "Other's Loop")

		status, _ := te.do(t, "POST", "/api/v1/api-keys", map[string]interface{}{
			"name":          "Bad Scope",
			"allowed_loops": []string{otherLoop},
		}, jwt)
		if status != 400 {
			t.Errorf("expected 400 for non-member loop, got %d", status)
		}
	})

	t.Run("allowed_loops in list response", func(t *testing.T) {
		status, env := te.do(t, "GET", "/api/v1/api-keys", nil, jwt)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data struct {
			APIKeys []struct {
				AllowedLoops []string `json:"allowed_loops"`
			} `json:"api_keys"`
		}
		json.Unmarshal(env.Data, &data)
		found := false
		for _, k := range data.APIKeys {
			if len(k.AllowedLoops) == 1 && k.AllowedLoops[0] == loop1 {
				found = true
			}
		}
		if !found {
			t.Error("scoped key not found in list response")
		}
	})

	t.Run("scoped key can access allowed loop", func(t *testing.T) {
		key, _ := te.createAPIKeyWithOptions(t, jwt, "Allowed", nil, []string{loop1})
		reqID := te.createRequest(t, key, loop1, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})
		if reqID == "" {
			t.Fatal("expected request to be created in allowed loop")
		}
	})

	t.Run("scoped key blocked from disallowed loop", func(t *testing.T) {
		key, _ := te.createAPIKeyWithOptions(t, jwt, "Restricted", nil, []string{loop1})
		status, _ := te.do(t, "POST", fmt.Sprintf("/api/v1/loops/%s/requests", loop2), map[string]interface{}{
			"processing_type": "time-sensitive",
			"type":            "markdown",
			"title":           "Should Fail",
			"request_text":    "should fail",
			"response_type":   "boolean",
			"response_config": map[string]interface{}{"true_label": "Y", "false_label": "N"},
			"timeout_seconds": 3600,
		}, key)
		if status != 403 {
			t.Errorf("expected 403 for disallowed loop, got %d", status)
		}
	})

	t.Run("unscoped key can access any loop", func(t *testing.T) {
		key := te.createAPIKey(t, jwt)
		req1 := te.createRequest(t, key, loop1, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})
		req2 := te.createRequest(t, key, loop2, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})
		if req1 == "" || req2 == "" {
			t.Fatal("unscoped key should access both loops")
		}
	})
}

func TestAPIKeys_LoopScopingOnGet(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "scope-get@test.dev", "Scope Get User")
	loop1, _ := te.createLoop(t, jwt, "Get Loop A")
	loop2, _ := te.createLoop(t, jwt, "Get Loop B")

	// Create requests in both loops
	unscopedKey := te.createAPIKey(t, jwt)
	req1 := te.createRequest(t, unscopedKey, loop1, "boolean", map[string]interface{}{
		"true_label": "Y", "false_label": "N",
	})
	req2 := te.createRequest(t, unscopedKey, loop2, "boolean", map[string]interface{}{
		"true_label": "Y", "false_label": "N",
	})

	// Scoped key to loop1 only
	scopedKey, _ := te.createAPIKeyWithOptions(t, jwt, "Scoped", nil, []string{loop1})

	t.Run("scoped key can get request in allowed loop", func(t *testing.T) {
		status, _ := te.do(t, "GET", "/api/v1/requests/"+req1, nil, scopedKey)
		if status != 200 {
			t.Errorf("expected 200, got %d", status)
		}
	})

	t.Run("scoped key blocked from getting request in disallowed loop", func(t *testing.T) {
		status, _ := te.do(t, "GET", "/api/v1/requests/"+req2, nil, scopedKey)
		if status != 403 {
			t.Errorf("expected 403, got %d", status)
		}
	})

	t.Run("scoped key blocked from cancelling request in disallowed loop", func(t *testing.T) {
		status, _ := te.do(t, "DELETE", "/api/v1/requests/"+req2, nil, scopedKey)
		if status != 403 {
			t.Errorf("expected 403, got %d", status)
		}
	})
}

func TestAPIKeys_AgentNameOnRequest(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "agent-req@test.dev", "Agent Req User")
	loopID, _ := te.createLoop(t, jwt, "Agent Req Loop")

	agent := "Claude"
	key, _ := te.createAPIKeyWithOptions(t, jwt, "Agent Key", &agent, nil)

	t.Run("agent_name propagated to created request", func(t *testing.T) {
		reqID := te.createRequest(t, key, loopID, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})

		status, env := te.do(t, "GET", "/api/v1/requests/"+reqID, nil, jwt)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data struct {
			Request struct {
				AgentName *string `json:"agent_name"`
			} `json:"request"`
		}
		json.Unmarshal(env.Data, &data)
		if data.Request.AgentName == nil || *data.Request.AgentName != "Claude" {
			t.Errorf("agent_name = %v, want 'Claude'", data.Request.AgentName)
		}
	})

	t.Run("no agent_name when using session auth", func(t *testing.T) {
		reqID := te.createRequest(t, jwt, loopID, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})

		status, env := te.do(t, "GET", "/api/v1/requests/"+reqID, nil, jwt)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data struct {
			Request struct {
				AgentName *string `json:"agent_name"`
			} `json:"request"`
		}
		json.Unmarshal(env.Data, &data)
		if data.Request.AgentName != nil {
			t.Errorf("agent_name = %v, want nil for session-created request", data.Request.AgentName)
		}
	})

	t.Run("no agent_name when key has no agent_name", func(t *testing.T) {
		plainKey := te.createAPIKey(t, jwt)
		reqID := te.createRequest(t, plainKey, loopID, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})

		status, env := te.do(t, "GET", "/api/v1/requests/"+reqID, nil, jwt)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data struct {
			Request struct {
				AgentName *string `json:"agent_name"`
			} `json:"request"`
		}
		json.Unmarshal(env.Data, &data)
		if data.Request.AgentName != nil {
			t.Errorf("agent_name = %v, want nil for key without agent_name", data.Request.AgentName)
		}
	})

	t.Run("agent_name visible in list requests", func(t *testing.T) {
		te.createRequest(t, key, loopID, "boolean", map[string]interface{}{
			"true_label": "Y", "false_label": "N",
		})

		status, env := te.do(t, "GET", "/api/v1/requests", nil, jwt)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data struct {
			Requests []struct {
				AgentName *string `json:"agent_name"`
			} `json:"requests"`
		}
		json.Unmarshal(env.Data, &data)
		found := false
		for _, r := range data.Requests {
			if r.AgentName != nil && *r.AgentName == "Claude" {
				found = true
			}
		}
		if !found {
			t.Error("agent_name 'Claude' not found in list results")
		}
	})
}

func TestAPIKeys_LoopScopingOnListLoops(t *testing.T) {
	te := setupTestEnv(t)
	jwt := te.devLogin(t, "scope-loops@test.dev", "Scope Loops User")
	loop1, _ := te.createLoop(t, jwt, "Visible Loop")
	te.createLoop(t, jwt, "Hidden Loop")

	key, _ := te.createAPIKeyWithOptions(t, jwt, "Scoped Loops", nil, []string{loop1})

	t.Run("scoped key only sees allowed loops", func(t *testing.T) {
		status, env := te.do(t, "GET", "/api/v1/loops", nil, key)
		if status != 200 {
			t.Fatalf("expected 200, got %d", status)
		}
		var data struct {
			Loops []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"loops"`
			Count int `json:"count"`
		}
		json.Unmarshal(env.Data, &data)
		if data.Count != 1 {
			t.Errorf("count = %d, want 1", data.Count)
		}
		if len(data.Loops) != 1 || data.Loops[0].ID != loop1 {
			t.Errorf("loops = %v, want only loop %s", data.Loops, loop1)
		}
	})
}
