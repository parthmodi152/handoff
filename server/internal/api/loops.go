package api

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/models"
)

const inviteCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// LoopHandler handles loop management endpoints.
type LoopHandler struct {
	db *db.DB
}

func NewLoopHandler(database *db.DB) *LoopHandler {
	return &LoopHandler{db: database}
}

// CreateLoop creates a new loop.
func (h *LoopHandler) CreateLoop(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	}
	if !DecodeJSONBody(w, r, &input) {
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 100 {
		writeError(w, http.StatusBadRequest, "name is required and must be 1-100 characters")
		return
	}
	if len(input.Description) > 500 {
		writeError(w, http.StatusBadRequest, "description must be 0-500 characters")
		return
	}
	if input.Icon == "" {
		input.Icon = "circle"
	}

	inviteCode, err := GenerateUniqueInviteCode(r.Context(), h.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate invite code")
		return
	}

	now := models.NowUTC()
	loop := &models.Loop{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Description: input.Description,
		Icon:        input.Icon,
		CreatorID:   user.ID,
		InviteCode:  inviteCode,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.db.CreateLoop(r.Context(), loop); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create loop")
		return
	}

	// Generate QR code
	qrPNG, err := qrcode.Encode(inviteCode, qrcode.Medium, 256)
	if err != nil {
		slog.Error("failed to generate QR code", "error", err)
	}
	qrBase64 := ""
	if qrPNG != nil {
		qrBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG)
	}

	writeSuccess(w, http.StatusCreated, "Loop created", map[string]interface{}{
		"loop": map[string]interface{}{
			"id":           loop.ID,
			"name":         loop.Name,
			"description":  loop.Description,
			"icon":         loop.Icon,
			"creator_id":   loop.CreatorID,
			"invite_code":  loop.InviteCode,
			"invite_qr":    qrBase64,
			"member_count": 1,
			"created_at":   loop.CreatedAt,
			"updated_at":   loop.UpdatedAt,
		},
	})
}

// ListLoops lists loops the user belongs to.
func (h *LoopHandler) ListLoops(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	loops, err := h.db.ListLoops(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list loops")
		return
	}

	if loops == nil {
		loops = []models.LoopWithMeta{}
	}

	// Filter by API key's allowed loops
	apiKey := GetAPIKeyFromContext(r)
	if apiKey != nil && len(apiKey.AllowedLoops) > 0 {
		allowed := make(map[string]bool, len(apiKey.AllowedLoops))
		for _, id := range apiKey.AllowedLoops {
			allowed[id] = true
		}
		filtered := loops[:0]
		for _, l := range loops {
			if allowed[l.ID] {
				filtered = append(filtered, l)
			}
		}
		loops = filtered
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"loops": loops,
		"count": len(loops),
	})
}

// GetLoop gets a single loop with its members.
func (h *LoopHandler) GetLoop(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	loopID := RequirePathParam(w, r, "id")
	if loopID == "" {
		return
	}

	if !EnforceLoopAccess(w, r, loopID) {
		return
	}

	loop := FetchLoopOr404(w, r, h.db, loopID)
	if loop == nil {
		return
	}

	if !EnforceMembership(w, r, h.db, loopID, user.ID) {
		return
	}

	members, err := h.db.GetLoopMembers(r.Context(), loopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get members")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"loop":    loop,
		"members": members,
	})
}

// JoinLoop joins a loop by invite code.
func (h *LoopHandler) JoinLoop(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	var input struct {
		InviteCode string `json:"invite_code"`
	}
	if !DecodeJSONBody(w, r, &input) {
		return
	}

	input.InviteCode = strings.TrimSpace(strings.ToUpper(input.InviteCode))
	if len(input.InviteCode) != 6 {
		writeError(w, http.StatusBadRequest, "invite_code must be exactly 6 characters")
		return
	}

	loop, err := h.db.GetLoopByInviteCode(r.Context(), input.InviteCode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to look up invite code")
		return
	}
	if loop == nil {
		writeError(w, http.StatusNotFound, "No loop found with that invite code")
		return
	}

	// Check if already member
	isMember, _, err := h.db.IsLoopMember(r.Context(), loop.ID, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check membership")
		return
	}
	if isMember {
		writeError(w, http.StatusConflict, "You are already a member of this loop")
		return
	}

	if err := h.db.JoinLoop(r.Context(), loop.ID, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to join loop")
		return
	}

	memberCount, _ := h.db.GetLoopMemberCount(r.Context(), loop.ID)

	writeSuccess(w, http.StatusOK, "Joined loop", map[string]interface{}{
		"loop": map[string]interface{}{
			"id":           loop.ID,
			"name":         loop.Name,
			"description":  loop.Description,
			"icon":         loop.Icon,
			"role":         "member",
			"member_count": memberCount,
		},
	})
}

func generateInviteCode() string {
	code := make([]byte, 6)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
		code[i] = inviteCodeChars[n.Int64()]
	}
	return string(code)
}
