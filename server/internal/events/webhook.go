package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// WebhookPayload is the JSON body sent to callback URLs.
type WebhookPayload struct {
	Event     string          `json:"event"`      // e.g. "request.completed"
	RequestID string          `json:"request_id"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// WebhookSender fires HTTP POST requests to callback URLs.
type WebhookSender struct {
	client *http.Client
}

// NewWebhookSender creates a webhook sender with a 10-second timeout.
func NewWebhookSender() *WebhookSender {
	return &WebhookSender{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send POSTs the event to the callback URL with retry (3 attempts, 1s/2s/4s backoff for 5xx).
func (ws *WebhookSender) Send(ctx context.Context, callbackURL string, event Event) error {
	payload := WebhookPayload{
		Event:     "request." + event.Status,
		RequestID: event.RequestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      event.Data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create webhook request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := ws.client.Do(req)
		if err != nil {
			lastErr = err
			slog.Warn("webhook attempt failed", "attempt", attempt+1, "url", callbackURL, "error", err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("webhook returned %d", resp.StatusCode)
			slog.Warn("webhook attempt got server error", "attempt", attempt+1, "url", callbackURL, "status", resp.StatusCode)
			continue
		}

		// 4xx — don't retry
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}

	return fmt.Errorf("webhook failed after 3 attempts: %w", lastErr)
}
