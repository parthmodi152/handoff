package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/models"
)

// Envelope is the standard response envelope.
type Envelope struct {
	Error bool        `json:"error"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	env := Envelope{Error: false, Msg: "Success", Data: data}
	writeEnvelope(w, status, env)
}

func writeSuccess(w http.ResponseWriter, status int, msg string, data interface{}) {
	env := Envelope{Error: false, Msg: msg, Data: data}
	writeEnvelope(w, status, env)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	env := Envelope{Error: true, Msg: msg, Data: nil}
	writeEnvelope(w, status, env)
}

func writeEnvelope(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(env); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

func decodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// RequireUser extracts the authenticated user from context.
// Returns nil and writes a 401 if unauthenticated.
func RequireUser(w http.ResponseWriter, r *http.Request) *models.User {
	user := GetUserFromContext(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
	}
	return user
}

// RequirePathParam extracts a path parameter and writes 400 if empty.
// Returns empty string on failure (response already written).
func RequirePathParam(w http.ResponseWriter, r *http.Request, param string) string {
	val := r.PathValue(param)
	if val == "" {
		writeError(w, http.StatusBadRequest, "Missing "+param)
	}
	return val
}

// FetchRequestOr404 loads a request by ID, writing errors on failure.
// Returns nil if not found or on error (response already written).
func FetchRequestOr404(w http.ResponseWriter, r *http.Request, database *db.DB, id string) *models.Request {
	req, err := database.GetRequest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get request")
		return nil
	}
	if req == nil {
		writeError(w, http.StatusNotFound, "Request not found")
		return nil
	}
	return req
}

// EnforceMembership checks loop membership and writes a 403 if denied.
// Returns true if the user is a member.
func EnforceMembership(w http.ResponseWriter, r *http.Request, database *db.DB, loopID, userID string) bool {
	isMember, _, err := database.IsLoopMember(r.Context(), loopID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check membership")
		return false
	}
	if !isMember {
		writeError(w, http.StatusForbidden, "You are not a member of this loop")
		return false
	}
	return true
}

// ResolveLoopName looks up a loop's display name, falling back to loopID.
func ResolveLoopName(database *db.DB, loopID string) string {
	loop, err := database.GetLoop(context.Background(), loopID)
	if err == nil && loop != nil {
		return loop.Name
	}
	return loopID
}

// RequireDashboardUser extracts the user and redirects to /login if absent.
// Returns nil when redirected (response already written).
func RequireDashboardUser(w http.ResponseWriter, r *http.Request) *models.User {
	user := GetUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	return user
}

// FetchLoopOr404 loads a loop by ID, writing errors on failure.
// Returns nil if not found or on error (response already written).
func FetchLoopOr404(w http.ResponseWriter, r *http.Request, database *db.DB, id string) *models.Loop {
	loop, err := database.GetLoop(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get loop")
		return nil
	}
	if loop == nil {
		writeError(w, http.StatusNotFound, "Loop not found")
		return nil
	}
	return loop
}

// DecodeJSONBody decodes a JSON request body and writes 400 on failure.
// Returns false if decoding failed (response already written).
func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := decodeJSON(r, dst); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return false
	}
	return true
}

// GenerateUniqueInviteCode generates a unique 6-character invite code,
// retrying up to 10 times to avoid collisions.
func GenerateUniqueInviteCode(ctx context.Context, database *db.DB) (string, error) {
	for i := 0; i < 10; i++ {
		code := generateInviteCode()
		exists, err := database.InviteCodeExists(ctx, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique invite code after 10 attempts")
}

// GenerateRawAPIKey creates a new hnd_-prefixed key and its bcrypt hash.
func GenerateRawAPIKey() (rawKey, hash string, err error) {
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	rawKey = "hnd_" + hex.EncodeToString(randomBytes)
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return rawKey, string(hashBytes), nil
}

// FetchLoopOrError loads a loop by ID for dashboard pages, using http.Error.
// Returns nil if not found or on error (response already written).
func FetchLoopOrError(w http.ResponseWriter, r *http.Request, database *db.DB, id string) *models.Loop {
	loop, err := database.GetLoop(r.Context(), id)
	if err != nil {
		slog.Error("failed to get loop", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil
	}
	if loop == nil {
		http.Error(w, "Loop not found", http.StatusNotFound)
		return nil
	}
	return loop
}

// FetchRequestOrError loads a request by ID for dashboard pages, using http.Error.
// Returns nil if not found or on error (response already written).
func FetchRequestOrError(w http.ResponseWriter, r *http.Request, database *db.DB, id string) *models.Request {
	req, err := database.GetRequest(r.Context(), id)
	if err != nil {
		slog.Error("failed to get request", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil
	}
	if req == nil {
		http.Error(w, "Request not found", http.StatusNotFound)
		return nil
	}
	return req
}

// EnforceDashboardMembership checks loop membership for dashboard pages.
// Returns true if the user is a member.
func EnforceDashboardMembership(w http.ResponseWriter, r *http.Request, database *db.DB, loopID, userID string) bool {
	isMember, _, err := database.IsLoopMember(r.Context(), loopID, userID)
	if err != nil {
		slog.Error("failed to check membership", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return false
	}
	if !isMember {
		http.Error(w, "You are not a member of this loop", http.StatusForbidden)
		return false
	}
	return true
}

// NewAPIKeyModel builds an APIKey model from the common fields.
func NewAPIKeyModel(userID, name, keyHash, keyPrefix string, permissions []string, agentName *string, allowedLoops []string) *models.APIKey {
	return &models.APIKey{
		ID:           uuid.New().String(),
		UserID:       userID,
		Name:         name,
		KeyHash:      keyHash,
		KeyPrefix:    keyPrefix,
		Permissions:  permissions,
		AgentName:    agentName,
		AllowedLoops: allowedLoops,
		IsActive:     true,
		CreatedAt:    models.NowUTC(),
	}
}
