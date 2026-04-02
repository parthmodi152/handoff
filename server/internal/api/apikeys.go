package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/models"
)

// APIKeyHandler handles API key management endpoints.
type APIKeyHandler struct {
	db *db.DB
}

func NewAPIKeyHandler(database *db.DB) *APIKeyHandler {
	return &APIKeyHandler{db: database}
}

// CreateAPIKey creates a new API key.
func (h *APIKeyHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	var input struct {
		Name         string   `json:"name"`
		AgentName    *string  `json:"agent_name"`
		Permissions  []string `json:"permissions"`
		AllowedLoops []string `json:"allowed_loops"`
		ExpiresAt    *string  `json:"expires_at"`
	}
	if !DecodeJSONBody(w, r, &input) {
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 100 {
		writeError(w, http.StatusBadRequest, "name is required and must be 1-100 characters")
		return
	}

	// Validate agent_name
	if input.AgentName != nil {
		trimmed := strings.TrimSpace(*input.AgentName)
		if trimmed == "" {
			input.AgentName = nil
		} else if len(trimmed) > 100 {
			writeError(w, http.StatusBadRequest, "agent_name must be 1-100 characters")
			return
		} else {
			input.AgentName = &trimmed
		}
	}

	// Validate allowed_loops — each must exist and user must be a member
	for _, loopID := range input.AllowedLoops {
		isMember, _, err := h.db.IsLoopMember(r.Context(), loopID, user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to validate loop access")
			return
		}
		if !isMember {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("You are not a member of loop %s", loopID))
			return
		}
	}

	// Default permissions
	if len(input.Permissions) == 0 {
		input.Permissions = []string{"loops:read", "loops:write", "requests:read", "requests:write"}
	}

	// Validate permissions
	validPerms := map[string]bool{
		"loops:read": true, "loops:write": true,
		"requests:read": true, "requests:write": true,
		"webhooks:read": true, "webhooks:write": true,
		"api_keys:read": true, "api_keys:write": true,
	}
	for _, p := range input.Permissions {
		if !validPerms[p] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid permission: %s", p))
			return
		}
	}

	rawKey, keyHash, err := GenerateRawAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate key")
		return
	}

	apiKey := NewAPIKeyModel(user.ID, input.Name, keyHash, rawKey[:8], input.Permissions, input.AgentName, input.AllowedLoops)
	apiKey.ExpiresAt = input.ExpiresAt

	if err := h.db.CreateAPIKey(r.Context(), apiKey); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create API key")
		return
	}

	resp := map[string]interface{}{
		"id":            apiKey.ID,
		"name":          apiKey.Name,
		"key":           rawKey,
		"key_prefix":    apiKey.KeyPrefix,
		"permissions":   apiKey.Permissions,
		"is_active":     true,
		"created_at":    apiKey.CreatedAt,
		"expires_at":    apiKey.ExpiresAt,
	}
	if apiKey.AgentName != nil {
		resp["agent_name"] = *apiKey.AgentName
	}
	if len(apiKey.AllowedLoops) > 0 {
		resp["allowed_loops"] = apiKey.AllowedLoops
	}

	writeSuccess(w, http.StatusCreated, "API key created", map[string]interface{}{
		"api_key": resp,
	})
}

// ListAPIKeys lists the user's API keys.
func (h *APIKeyHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	keys, err := h.db.ListAPIKeys(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list API keys")
		return
	}

	if keys == nil {
		keys = []models.APIKey{}
	}

	// Build response without sensitive fields
	keyList := make([]map[string]interface{}, len(keys))
	for i, k := range keys {
		keyList[i] = map[string]interface{}{
			"id":            k.ID,
			"name":          k.Name,
			"key_prefix":    k.KeyPrefix,
			"permissions":   k.Permissions,
			"agent_name":    k.AgentName,
			"allowed_loops": k.AllowedLoops,
			"is_active":     k.IsActive,
			"last_used_at":  k.LastUsedAt,
			"created_at":    k.CreatedAt,
			"expires_at":    k.ExpiresAt,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"api_keys": keyList,
		"count":    len(keyList),
	})
}

// RevokeAPIKey revokes an API key.
func (h *APIKeyHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	keyID := RequirePathParam(w, r, "id")
	if keyID == "" {
		return
	}

	if err := h.db.RevokeAPIKey(r.Context(), keyID, user.ID); err != nil {
		writeError(w, http.StatusNotFound, "API key not found")
		return
	}

	writeSuccess(w, http.StatusOK, "API key revoked", nil)
}
