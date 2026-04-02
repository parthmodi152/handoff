package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hitl-sh/handoff-server/internal/api"
	"github.com/hitl-sh/handoff-server/internal/config"
	"github.com/hitl-sh/handoff-server/internal/db"
)

// testEnv holds a configured test server and helpers.
type testEnv struct {
	server *httptest.Server
	db     *db.DB
	cfg    *config.Config
}

// envelope is the standard JSON response wrapper.
type envelope struct {
	Error bool            `json:"error"`
	Msg   string          `json:"msg"`
	Data  json.RawMessage `json:"data"`
}

// setupTestEnvBase creates a DB + config without a server.
// Use this when you need to customize the router (e.g., inject a broker).
func setupTestEnvBase(t *testing.T) *testEnv {
	t.Helper()

	dbPath := t.TempDir() + "/test.db"
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var migrations []db.MigrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile("../../migrations/" + entry.Name())
		if err != nil {
			t.Fatalf("read migration %s: %v", entry.Name(), err)
		}
		migrations = append(migrations, db.MigrationFile{Name: entry.Name(), Content: string(content)})
	}
	if err := database.Migrate(migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		Port:      "0",
		DBPath:    dbPath,
		BaseURL:   "http://localhost",
		JWTSecret: "test-secret-key-for-testing-32b!",
		DevMode:   true,
	}

	t.Cleanup(func() {
		database.Close()
	})

	return &testEnv{db: database, cfg: cfg}
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	te := setupTestEnvBase(t)

	router := api.NewRouter(te.db, te.cfg, nil, nil, nil) // nil push/broker/webhook for tests
	server := httptest.NewServer(router)

	t.Cleanup(func() {
		server.Close()
	})

	te.server = server
	return te
}

// do sends an HTTP request and returns status code + parsed envelope.
func (te *testEnv) do(t *testing.T, method, path string, body interface{}, token string) (int, envelope) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, te.server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode, env
}

// dataField extracts a string field from nested envelope data.
func dataField(t *testing.T, env envelope, keys ...string) string {
	t.Helper()
	var m map[string]json.RawMessage
	json.Unmarshal(env.Data, &m)
	current := m
	for i, key := range keys {
		raw, ok := current[key]
		if !ok {
			t.Fatalf("missing key %q in data", key)
		}
		if i == len(keys)-1 {
			var s string
			if err := json.Unmarshal(raw, &s); err != nil {
				return string(raw)
			}
			return s
		}
		json.Unmarshal(raw, &current)
	}
	return ""
}

// devLogin creates a dev user and returns the JWT.
func (te *testEnv) devLogin(t *testing.T, email, name string) string {
	t.Helper()
	status, env := te.do(t, "POST", "/auth/dev-login", map[string]string{
		"email": email, "name": name,
	}, "")
	if status != 200 {
		t.Fatalf("dev login failed: %d %s", status, env.Msg)
	}
	return dataField(t, env, "token")
}

// createAPIKey creates an API key and returns the raw key.
func (te *testEnv) createAPIKey(t *testing.T, jwt string) string {
	t.Helper()
	status, env := te.do(t, "POST", "/api/v1/api-keys", map[string]string{"name": "test-key"}, jwt)
	if status != 201 {
		t.Fatalf("create api key failed: %d %s", status, env.Msg)
	}
	return dataField(t, env, "api_key", "key")
}

// createAPIKeyWithOptions creates an API key with agent name and/or loop scoping.
// Returns (raw_key, key_id).
func (te *testEnv) createAPIKeyWithOptions(t *testing.T, jwt string, name string, agentName *string, allowedLoops []string) (string, string) {
	t.Helper()
	body := map[string]interface{}{"name": name}
	if agentName != nil {
		body["agent_name"] = *agentName
	}
	if len(allowedLoops) > 0 {
		body["allowed_loops"] = allowedLoops
	}
	status, env := te.do(t, "POST", "/api/v1/api-keys", body, jwt)
	if status != 201 {
		t.Fatalf("create api key failed: %d %s", status, env.Msg)
	}
	return dataField(t, env, "api_key", "key"), dataField(t, env, "api_key", "id")
}

// createLoop creates a loop and returns (loop_id, invite_code).
func (te *testEnv) createLoop(t *testing.T, token, name string) (string, string) {
	t.Helper()
	status, env := te.do(t, "POST", "/api/v1/loops", map[string]interface{}{
		"name": name, "description": "Test loop", "icon": "shield",
	}, token)
	if status != 201 {
		t.Fatalf("create loop failed: %d %s", status, env.Msg)
	}
	return dataField(t, env, "loop", "id"), dataField(t, env, "loop", "invite_code")
}

// createRequest creates a request and returns its ID.
func (te *testEnv) createRequest(t *testing.T, token, loopID, respType string, respConfig map[string]interface{}) string {
	t.Helper()
	timeout := 3600
	body := map[string]interface{}{
		"processing_type": "time-sensitive",
		"type":            "markdown",
		"title":           fmt.Sprintf("Test %s Title", respType),
		"request_text":    fmt.Sprintf("test %s request", respType),
		"response_type":   respType,
		"response_config": respConfig,
		"timeout_seconds": timeout,
	}
	status, env := te.do(t, "POST", fmt.Sprintf("/api/v1/loops/%s/requests", loopID), body, token)
	if status != 201 {
		t.Fatalf("create request failed: %d %s", status, env.Msg)
	}
	return dataField(t, env, "request_id")
}
