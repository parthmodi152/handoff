package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// StreamRequestEvents opens an SSE connection that delivers a single event
// when the request's status changes (completed, cancelled, or timeout).
// If the request is already terminal when the client connects, the event
// is sent immediately and the connection closes.
//
// Endpoint: GET /api/v1/requests/{id}/events
func (h *RequestHandler) StreamRequestEvents(w http.ResponseWriter, r *http.Request) {
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

	// Verify access: must be creator or loop member
	if req.CreatorID != user.ID {
		if !EnforceMembership(w, r, h.db, req.LoopID, user.ID) {
			return
		}
	}

	// Check if http.Flusher is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	// Disable write deadline for this long-lived connection (Go 1.20+).
	// Ignore error — not all ResponseWriter implementations support deadlines (e.g. httptest).
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	// If request is already terminal, send event immediately and close
	if req.Status != "pending" {
		writeSSEEvent(w, flusher, req.Status, reqID, req)
		return
	}

	// Broker not configured — fall back to telling the client to poll
	if h.broker == nil {
		writeSSEComment(w, flusher, "broker not available, poll GET /api/v1/requests/"+reqID)
		return
	}

	// Subscribe and wait for event
	eventCh, unsubscribe := h.broker.Subscribe(reqID)
	defer unsubscribe()

	slog.Info("SSE client connected", "request_id", reqID, "user_id", user.ID)

	// Send initial comment to confirm connection
	writeSSEComment(w, flusher, "connected")

	select {
	case event := <-eventCh:
		writeSSEEventRaw(w, flusher, event.Status, event.Data)
		slog.Info("SSE event delivered", "request_id", reqID, "status", event.Status)

	case <-r.Context().Done():
		slog.Info("SSE client disconnected", "request_id", reqID)
	}
}

// writeSSEEvent formats and writes a request status event.
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, status, reqID string, req interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"request_id": reqID,
		"status":     status,
		"request":    req,
	})
	writeSSEEventRaw(w, flusher, status, data)
}

// writeSSEEventRaw writes pre-serialized event data in SSE format.
func writeSSEEventRaw(w http.ResponseWriter, flusher http.Flusher, eventType string, data json.RawMessage) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
	flusher.Flush()
}

// writeSSEComment writes an SSE comment line (used for keepalive or info).
func writeSSEComment(w http.ResponseWriter, flusher http.Flusher, comment string) {
	fmt.Fprintf(w, ": %s\n\n", comment)
	flusher.Flush()
}
