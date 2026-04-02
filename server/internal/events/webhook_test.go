package events

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestWebhookSend_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
	}))
	defer server.Close()

	ws := NewWebhookSender()
	event := Event{
		RequestID: "req-123",
		LoopID:    "loop-456",
		Status:    "completed",
		Data:      json.RawMessage(`{"request":{"id":"req-123"}}`),
	}

	err := ws.Send(context.Background(), server.URL, event)
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	// Verify payload format
	var payload WebhookPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Event != "request.completed" {
		t.Errorf("event = %q, want %q", payload.Event, "request.completed")
	}
	if payload.RequestID != "req-123" {
		t.Errorf("request_id = %q, want %q", payload.RequestID, "req-123")
	}
	if payload.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
}

func TestWebhookSend_ServerError_Retries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(500)
	}))
	defer server.Close()

	ws := NewWebhookSender()
	event := Event{
		RequestID: "req-123",
		Status:    "completed",
		Data:      json.RawMessage(`{}`),
	}

	err := ws.Send(context.Background(), server.URL, event)
	if err == nil {
		t.Fatal("expected error on 500")
	}

	// Should have retried 3 times
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestWebhookSend_ClientError_NoRetry(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(400)
	}))
	defer server.Close()

	ws := NewWebhookSender()
	event := Event{
		RequestID: "req-123",
		Status:    "completed",
		Data:      json.RawMessage(`{}`),
	}

	err := ws.Send(context.Background(), server.URL, event)
	if err == nil {
		t.Fatal("expected error on 400")
	}

	// Should NOT retry on 4xx
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on 4xx)", got)
	}
}

func TestWebhookSend_ContextCancel(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(500)
	}))
	defer server.Close()

	ws := NewWebhookSender()
	event := Event{
		RequestID: "req-123",
		Status:    "completed",
		Data:      json.RawMessage(`{}`),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := ws.Send(ctx, server.URL, event)
	if err == nil {
		t.Fatal("expected error on context cancel")
	}

	// Should have been cut short by context cancellation (1 attempt + partial backoff wait)
	if got := attempts.Load(); got > 2 {
		t.Errorf("attempts = %d, expected <= 2 (context should cancel during backoff)", got)
	}
}

func TestWebhookSend_NetworkError(t *testing.T) {
	ws := NewWebhookSender()
	event := Event{
		RequestID: "req-123",
		Status:    "completed",
		Data:      json.RawMessage(`{}`),
	}

	// Use a URL that won't connect
	err := ws.Send(context.Background(), "http://127.0.0.1:1", event)
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}
