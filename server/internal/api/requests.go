package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/events"
	"github.com/hitl-sh/handoff-server/internal/models"
	"github.com/hitl-sh/handoff-server/internal/push"
)

// RequestHandler handles request endpoints.
type RequestHandler struct {
	db      *db.DB
	push    push.Sender           // nil when APNs not configured
	broker  *events.Broker        // nil when event broker not configured
	webhook *events.WebhookSender // nil when webhooks not configured
}

func NewRequestHandler(database *db.DB, pushSender push.Sender, broker *events.Broker, webhook *events.WebhookSender) *RequestHandler {
	return &RequestHandler{db: database, push: pushSender, broker: broker, webhook: webhook}
}

// CreateRequest creates a new review request.
func (h *RequestHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}
	apiKey := GetAPIKeyFromContext(r)

	loopID := RequirePathParam(w, r, "loop_id")
	if loopID == "" {
		return
	}

	var input models.CreateRequestInput
	if !DecodeJSONBody(w, r, &input) {
		return
	}

	// Validate fields
	if err := validateCreateRequest(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate response config
	if err := validateResponseConfig(input.ResponseType, input.ResponseConfig); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid response_config: "+err.Error())
		return
	}

	// Validate default_response if provided
	if input.DefaultResponse != nil && len(input.DefaultResponse) > 0 && string(input.DefaultResponse) != "null" {
		if err := validateResponseData(input.ResponseType, input.ResponseConfig, input.DefaultResponse); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid default_response: "+err.Error())
			return
		}
	}

	if !EnforceLoopAccess(w, r, loopID) {
		return
	}

	if !EnforceMembership(w, r, h.db, loopID, user.ID) {
		return
	}

	now := models.NowUTC()
	reqID := uuid.New().String()

	var timeoutAt *string
	if input.TimeoutSeconds != nil {
		t, _ := time.Parse(time.RFC3339, now)
		ta := t.Add(time.Duration(*input.TimeoutSeconds) * time.Second).UTC().Format(time.RFC3339)
		timeoutAt = &ta
	}

	var apiKeyID *string
	if apiKey != nil {
		apiKeyID = &apiKey.ID
	}

	platform := input.Platform
	if platform == "" {
		platform = "api"
	}

	contentType := input.Type
	if contentType != "markdown" && contentType != "image" {
		contentType = "markdown"
	}

	priority := input.Priority
	if priority == "" {
		priority = "medium"
	}

	var agentName *string
	if apiKey != nil && apiKey.AgentName != nil {
		agentName = apiKey.AgentName
	}

	req := &models.Request{
		ID:              reqID,
		LoopID:          loopID,
		CreatorID:       user.ID,
		APIKeyID:        apiKeyID,
		AgentName:       agentName,
		ProcessingType:  input.ProcessingType,
		ContentType:     contentType,
		Priority:        priority,
		Title:           input.Title,
		RequestText:     input.RequestText,
		ImageURL:        input.ImageURL,
		Context:         input.Context,
		Platform:        platform,
		ResponseType:    input.ResponseType,
		ResponseConfig:  input.ResponseConfig,
		DefaultResponse: input.DefaultResponse,
		TimeoutSeconds:  input.TimeoutSeconds,
		TimeoutAt:       timeoutAt,
		CallbackURL:     input.CallbackURL,
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.db.CreateRequest(r.Context(), req); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}

	// Send push notifications to loop members
	memberCount, _ := h.db.GetActiveLoopMemberCount(r.Context(), loopID)
	var notificationsSent int
	if h.push != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic in push goroutine", "recover", r, "loop_id", loopID)
				}
			}()

			loopName := ResolveLoopName(h.db, loopID)
			var trueLabel, falseLabel string
			if req.ResponseType == "boolean" && len(req.ResponseConfig) > 0 {
				var cfg struct {
					TrueLabel  string `json:"true_label"`
					FalseLabel string `json:"false_label"`
				}
				if err := json.Unmarshal(req.ResponseConfig, &cfg); err == nil {
					trueLabel = cfg.TrueLabel
					falseLabel = cfg.FalseLabel
				}
			}
			ntf := push.NewRequestPayload(req.Title, req.RequestText, loopName, req.ResponseType, reqID, loopID, trueLabel, falseLabel)
			excludeUserID := user.ID
			if apiKey != nil {
				excludeUserID = "" // agent workflow — don't exclude the reviewer
			}
			sent, failed := h.push.NotifyLoopMembers(context.Background(), loopID, excludeUserID, ntf)
			slog.Info("push notifications sent", "loop_id", loopID, "sent", sent, "failed", failed)
		}()

		// Update Live Activities for this loop
		go func() {
			defer func() {
				if rv := recover(); rv != nil {
					slog.Error("panic in live activity goroutine", "recover", rv, "loop_id", loopID)
				}
			}()
			loopName := ResolveLoopName(h.db, loopID)
			push.RefreshLoopActivity(context.Background(), h.push, h.db, loopID,
				"New request in "+loopName, push.TruncateText(req.Title, 100))
		}()
	}

	writeSuccess(w, http.StatusCreated, "Request created", map[string]interface{}{
		"request_id":         reqID,
		"status":             "pending",
		"timeout_at":         timeoutAt,
		"broadcasted_to":     memberCount,
		"notifications_sent": notificationsSent,
	})
}

// ListRequests lists requests accessible to the user.
func (h *RequestHandler) ListRequests(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}

	q := r.URL.Query()
	status := q.Get("status")
	priority := q.Get("priority")
	loopID := q.Get("loop_id")
	sort := q.Get("sort")
	if sort == "" {
		sort = "created_at_desc"
	}

	limit := 50
	if v := q.Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l >= 1 && l <= 100 {
			limit = l
		}
	}
	offset := 0
	if v := q.Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
		}
	}

	requests, total, err := h.db.ListRequests(r.Context(), user.ID, status, priority, loopID, sort, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list requests")
		return
	}

	if requests == nil {
		requests = []models.Request{}
	}

	hasMore := offset+limit < total
	var nextOffset *int
	if hasMore {
		no := offset + limit
		nextOffset = &no
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"requests": requests,
		"count":    len(requests),
		"total":    total,
		"has_more": hasMore,
		"pagination": map[string]interface{}{
			"limit":       limit,
			"offset":      offset,
			"next_offset": nextOffset,
		},
	})
}

// GetRequest gets a single request.
func (h *RequestHandler) GetRequest(w http.ResponseWriter, r *http.Request) {
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

	if !EnforceLoopAccess(w, r, req.LoopID) {
		return
	}

	if !EnforceMembership(w, r, h.db, req.LoopID, user.ID) {
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"request": req,
	})
}

// CancelRequest cancels a pending request.
func (h *RequestHandler) CancelRequest(w http.ResponseWriter, r *http.Request) {
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

	if !EnforceLoopAccess(w, r, req.LoopID) {
		return
	}

	if req.Status != "pending" {
		writeError(w, http.StatusBadRequest, "Request is not pending")
		return
	}

	// Check if user is request creator or loop creator
	if req.CreatorID != user.ID {
		loop, err := h.db.GetLoop(r.Context(), req.LoopID)
		if err != nil || loop == nil || loop.CreatorID != user.ID {
			writeError(w, http.StatusForbidden, "Only the request creator or loop creator can cancel")
			return
		}
	}

	if err := h.db.CancelRequest(r.Context(), reqID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to cancel request")
		return
	}

	// Send cancellation push notifications
	if h.push != nil {
		go func() {
			defer func() {
				if rv := recover(); rv != nil {
					slog.Error("panic in push goroutine", "recover", rv, "request_id", reqID)
				}
			}()

			loopName := ResolveLoopName(h.db, req.LoopID)
			ntf := push.CancelledPayload(req.Title, req.RequestText, loopName, reqID, req.LoopID)
			sent, failed := h.push.NotifyLoopMembers(context.Background(), req.LoopID, user.ID, ntf)
			slog.Info("cancellation push sent", "request_id", reqID, "sent", sent, "failed", failed)
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

	// Publish event to SSE subscribers
	cancelData := map[string]interface{}{
		"request": map[string]interface{}{
			"id":     reqID,
			"status": "cancelled",
		},
	}
	if h.broker != nil {
		eventData, _ := json.Marshal(cancelData)
		h.broker.Publish(events.Event{
			RequestID: reqID,
			LoopID:    req.LoopID,
			Status:    "cancelled",
			Data:      eventData,
		})
	}

	// Fire webhook if configured
	if req.CallbackURL != nil && h.webhook != nil {
		go func() {
			defer func() { recover() }()
			eventData, _ := json.Marshal(cancelData)
			if err := h.webhook.Send(context.Background(), *req.CallbackURL, events.Event{
				RequestID: reqID,
				LoopID:    req.LoopID,
				Status:    "cancelled",
				Data:      eventData,
			}); err != nil {
				slog.Error("webhook failed", "callback_url", *req.CallbackURL, "error", err)
			}
		}()
	}

	writeSuccess(w, http.StatusOK, "Request cancelled", map[string]interface{}{
		"request": map[string]interface{}{
			"id":         reqID,
			"status":     "cancelled",
			"updated_at": models.NowUTC(),
		},
	})
}

// TestAPIKey validates the API key and returns info.
func (h *RequestHandler) TestAPIKey(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}
	apiKey := GetAPIKeyFromContext(r)

	data := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	}

	if apiKey != nil {
		data["api_key"] = map[string]interface{}{
			"id":          apiKey.ID,
			"name":        apiKey.Name,
			"key_prefix":  apiKey.KeyPrefix,
			"permissions": apiKey.Permissions,
			"expires_at":  apiKey.ExpiresAt,
		}
	}

	writeSuccess(w, http.StatusOK, "API key is valid", data)
}
