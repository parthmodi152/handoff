package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/hitl-sh/handoff-server/internal/events"
	"github.com/hitl-sh/handoff-server/internal/models"
	"github.com/hitl-sh/handoff-server/internal/push"
)

// RespondToRequest submits a response to a pending request.
func (h *RequestHandler) RespondToRequest(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	reqID := RequirePathParam(w, r, "id")
	if reqID == "" {
		return
	}

	req := FetchRequestOr404(w, r, h.db, reqID)
	if req == nil {
		return
	}

	if req.Status != "pending" {
		writeError(w, http.StatusBadRequest, "Request is no longer pending")
		return
	}

	if !EnforceMembership(w, r, h.db, req.LoopID, user.ID) {
		return
	}

	var input models.RespondInput
	if !DecodeJSONBody(w, r, &input) {
		return
	}

	// Validate response data
	if err := validateResponseData(req.ResponseType, req.ResponseConfig, input.ResponseData); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid response_data: "+err.Error())
		return
	}

	if err := h.db.RespondToRequest(r.Context(), reqID, user.ID, input.ResponseData); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to submit response")
		return
	}

	// Get updated request for response time
	updatedReq, _ := h.db.GetRequest(r.Context(), reqID)

	// Send push notification to request creator
	if h.push != nil {
		go func() {
			defer func() {
				if rv := recover(); rv != nil {
					slog.Error("panic in push goroutine", "recover", rv, "request_id", reqID)
				}
			}()

			loopName := ResolveLoopName(h.db, req.LoopID)
			ntf := push.CompletedPayload(req.Title, req.RequestText, loopName, user.Name, reqID, req.LoopID)
			sent, failed := h.push.NotifyUser(context.Background(), req.CreatorID, ntf)
			slog.Info("completion push sent", "request_id", reqID, "sent", sent, "failed", failed)
		}()

		// Update Live Activities for this loop
		go func() {
			defer func() {
				if rv := recover(); rv != nil {
					slog.Error("panic in live activity goroutine", "recover", rv, "request_id", reqID)
				}
			}()
			push.RefreshLoopActivity(context.Background(), h.push, h.db, req.LoopID, "", "")
		}()
	}

	responseData := map[string]interface{}{
		"request": map[string]interface{}{
			"id":     reqID,
			"status": "completed",
			"response_data": json.RawMessage(input.ResponseData),
			"response_by": map[string]interface{}{
				"user_id": user.ID,
				"name":    user.Name,
				"email":   user.Email,
			},
		},
	}
	if updatedReq != nil {
		responseData["request"].(map[string]interface{})["response_at"] = updatedReq.ResponseAt
		responseData["request"].(map[string]interface{})["response_time_seconds"] = updatedReq.ResponseTimeSeconds
	}

	// Publish event to SSE subscribers
	if h.broker != nil {
		eventData, _ := json.Marshal(responseData)
		h.broker.Publish(events.Event{
			RequestID: reqID,
			LoopID:    req.LoopID,
			Status:    "completed",
			Data:      eventData,
		})
	}

	// Fire webhook if configured
	if req.CallbackURL != nil && h.webhook != nil {
		go func() {
			defer func() { recover() }()
			eventData, _ := json.Marshal(responseData)
			if err := h.webhook.Send(context.Background(), *req.CallbackURL, events.Event{
				RequestID: reqID,
				LoopID:    req.LoopID,
				Status:    "completed",
				Data:      eventData,
			}); err != nil {
				slog.Error("webhook failed", "callback_url", *req.CallbackURL, "error", err)
			}
		}()
	}

	writeSuccess(w, http.StatusOK, "Response submitted", responseData)
}
