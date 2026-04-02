package api

import (
	"log/slog"
	"net/http"

	"github.com/hitl-sh/handoff-server/internal/db"
)

// DeviceHandler handles device token registration endpoints.
type DeviceHandler struct {
	db *db.DB
}

// NewDeviceHandler creates a new DeviceHandler.
func NewDeviceHandler(database *db.DB) *DeviceHandler {
	return &DeviceHandler{db: database}
}

type registerDeviceInput struct {
	Token      string `json:"token"`
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
	LoopID     string `json:"loop_id"`
}

// RegisterDevice registers or updates a device token for push notifications.
// POST /api/v1/devices
func (h *DeviceHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	var input registerDeviceInput
	if !DecodeJSONBody(w, r, &input) {
		return
	}

	if input.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	platform := input.Platform
	if platform == "" {
		platform = "ios"
	}
	if platform != "ios" && platform != "macos" && platform != "ios_activity" {
		writeError(w, http.StatusBadRequest, "platform must be 'ios', 'macos', or 'ios_activity'")
		return
	}

	if platform == "ios_activity" {
		if input.LoopID == "" {
			writeError(w, http.StatusBadRequest, "loop_id is required for ios_activity platform")
			return
		}
		if err := h.db.UpsertActivityToken(r.Context(), user.ID, input.LoopID, input.Token); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to register activity token")
			return
		}
	} else {
		if err := h.db.UpsertDeviceToken(r.Context(), user.ID, input.Token, platform, input.AppVersion); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to register device token")
			return
		}
	}

	writeSuccess(w, http.StatusOK, "Device token registered", map[string]interface{}{
		"token":    input.Token,
		"platform": platform,
	})
}

// UnregisterDevice removes a device token.
// DELETE /api/v1/devices/{token}
func (h *DeviceHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}
	_ = user // used for auth only

	token := RequirePathParam(w, r, "token")
	if token == "" {
		return
	}

	// Try both tables — token could be in either
	if err := h.db.DeleteActivityToken(r.Context(), token); err != nil {
		slog.Warn("failed to delete activity token during unregister", "token", token, "error", err)
	}
	if err := h.db.DeleteDeviceToken(r.Context(), token); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to unregister device token")
		return
	}

	writeSuccess(w, http.StatusOK, "Device token unregistered", nil)
}
