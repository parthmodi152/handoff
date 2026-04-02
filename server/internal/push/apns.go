package push

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"

	"github.com/hitl-sh/handoff-server/internal/config"
)

// Client wraps an APNs client with retry logic and stale token cleanup.
type Client struct {
	apns  *apns2.Client
	topic string // bundle ID (e.g., "sh.hitl.handoff")
}

// NewClient creates an APNs client from config.
// Returns nil, nil if APNSKeyPath is empty (graceful degradation).
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.APNSKeyPath == "" {
		slog.Warn("push notifications disabled: HANDOFF_APNS_KEY_PATH not configured")
		return nil, nil
	}

	authKey, err := token.AuthKeyFromFile(cfg.APNSKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load APNs auth key from %s: %w", cfg.APNSKeyPath, err)
	}

	authToken := &token.Token{
		AuthKey: authKey,
		KeyID:   cfg.APNSKeyID,
		TeamID:  cfg.AppleTeamID,
	}

	var client *apns2.Client
	if cfg.APNSProduction {
		client = apns2.NewTokenClient(authToken).Production()
	} else {
		client = apns2.NewTokenClient(authToken).Development()
	}

	slog.Info("APNs push client initialized",
		"production", cfg.APNSProduction,
		"key_id", cfg.APNSKeyID,
		"team_id", cfg.AppleTeamID,
		"bundle_id", cfg.AppleBundleID,
	)

	return &Client{
		apns:  client,
		topic: cfg.AppleBundleID,
	}, nil
}

// Send sends a push notification to a single device token with retry.
// Returns the APNs response reason on permanent failure, or an error on transient failure after retries.
func (c *Client) Send(ctx context.Context, deviceToken string, p *payload.Payload) (*apns2.Response, error) {
	notification := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       c.topic,
		Payload:     p,
		Priority:    apns2.PriorityHigh,
	}
	return c.sendNotification(ctx, notification)
}

// SendLiveActivity sends a Live Activity push notification with a raw JSON payload.
func (c *Client) SendLiveActivity(ctx context.Context, activityToken string, rawPayload []byte) (*apns2.Response, error) {
	notification := &apns2.Notification{
		DeviceToken: activityToken,
		Topic:       c.topic + ".push-type.liveactivity",
		Payload:     rawPayload,
		Priority:    apns2.PriorityHigh,
		PushType:    apns2.PushTypeLiveActivity,
	}
	return c.sendNotification(ctx, notification)
}

// sendNotification sends a notification with retry logic (3 attempts, exponential backoff).
func (c *Client) sendNotification(ctx context.Context, notification *apns2.Notification) (*apns2.Response, error) {
	var lastResp *apns2.Response
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s
			select {
			case <-ctx.Done():
				return lastResp, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.apns.Push(notification)
		if err != nil {
			lastErr = err
			slog.Warn("APNs push network error, retrying",
				"attempt", attempt+1,
				"token_prefix", truncateToken(notification.DeviceToken),
				"error", err,
			)
			continue
		}

		lastResp = resp

		switch {
		case resp.StatusCode == 200:
			return resp, nil

		case resp.StatusCode == 410:
			// Gone — device uninstalled app. Don't retry.
			return resp, nil

		case resp.StatusCode == 429 || resp.StatusCode >= 500:
			// Transient — retry
			slog.Warn("APNs transient error, retrying",
				"attempt", attempt+1,
				"status", resp.StatusCode,
				"reason", resp.Reason,
				"token_prefix", truncateToken(notification.DeviceToken),
			)
			continue

		default:
			// Permanent failure (400, 401, 403, 404) — don't retry
			slog.Error("APNs permanent error",
				"status", resp.StatusCode,
				"reason", resp.Reason,
				"token_prefix", truncateToken(notification.DeviceToken),
			)
			return resp, nil
		}
	}

	if lastErr != nil {
		return lastResp, fmt.Errorf("APNs push failed after 3 retries: %w", lastErr)
	}
	return lastResp, fmt.Errorf("APNs push failed after 3 retries: status %d reason %s", lastResp.StatusCode, lastResp.Reason)
}

func truncateToken(token string) string {
	if len(token) > 8 {
		return token[:8] + "..."
	}
	return token
}
