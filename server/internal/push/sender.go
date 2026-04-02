package push

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"

	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/models"
)

const maxConcurrentPushes = 10

// SenderService implements the Sender interface using an APNs client.
type SenderService struct {
	client *Client
	db     *db.DB
}

// NewSenderService creates a new push notification sender.
func NewSenderService(client *Client, database *db.DB) *SenderService {
	return &SenderService{
		client: client,
		db:     database,
	}
}

// NotifyLoopMembers sends a push to all devices of all members in a loop,
// excluding the specified user.
func (s *SenderService) NotifyLoopMembers(ctx context.Context, loopID, excludeUserID string, p *payload.Payload) (sent, failed int) {
	tokens, err := s.db.GetDeviceTokensByLoopMembers(ctx, loopID, excludeUserID)
	if err != nil {
		slog.Error("failed to get device tokens for loop members",
			"loop_id", loopID,
			"error", err,
		)
		return 0, 0
	}

	if len(tokens) == 0 {
		return 0, 0
	}

	return s.sendToTokens(ctx, tokens, p)
}

// NotifyUser sends a push to all devices of a specific user.
func (s *SenderService) NotifyUser(ctx context.Context, userID string, p *payload.Payload) (sent, failed int) {
	tokens, err := s.db.GetDeviceTokensByUser(ctx, userID)
	if err != nil {
		slog.Error("failed to get device tokens for user",
			"user_id", userID,
			"error", err,
		)
		return 0, 0
	}

	if len(tokens) == 0 {
		return 0, 0
	}

	return s.sendToTokens(ctx, tokens, p)
}

// sendToTokens sends the same payload to multiple device tokens concurrently.
func (s *SenderService) sendToTokens(ctx context.Context, tokens []models.DeviceToken, p *payload.Payload) (sent, failed int) {
	tokenStrs := make([]string, len(tokens))
	for i, dt := range tokens {
		tokenStrs[i] = dt.Token
	}
	return s.fanOutSend(ctx, tokenStrs,
		func(ctx context.Context, tok string) (*apns2.Response, error) {
			return s.client.Send(ctx, tok, p)
		},
		func(ctx context.Context, tok string) {
			if delErr := s.db.DeleteDeviceToken(ctx, tok); delErr != nil {
				slog.Error("failed to delete stale device token",
					"token_prefix", truncateToken(tok),
					"error", delErr,
				)
			} else {
				slog.Info("deleted stale device token (410 Gone)",
					"token_prefix", truncateToken(tok),
				)
			}
		},
	)
}

// UpdateLoopActivity sends a Live Activity "update" push to all activity tokens for a loop.
func (s *SenderService) UpdateLoopActivity(ctx context.Context, loopID string, contentState interface{}, staleDateUnix *int64, alertTitle, alertBody string) (sent, failed int) {
	tokens, err := s.db.GetActivityTokensByLoop(ctx, loopID)
	if err != nil || len(tokens) == 0 {
		return 0, 0
	}
	payloadBytes, err := BuildLiveActivityPayload(contentState, staleDateUnix, alertTitle, alertBody)
	if err != nil {
		slog.Error("failed to build live activity payload", "error", err)
		return 0, 0
	}
	return s.sendToActivityTokens(ctx, tokens, payloadBytes)
}

// EndLoopActivity sends a Live Activity "end" push to all activity tokens for a loop.
func (s *SenderService) EndLoopActivity(ctx context.Context, loopID string, contentState interface{}) (sent, failed int) {
	tokens, err := s.db.GetActivityTokensByLoop(ctx, loopID)
	if err != nil || len(tokens) == 0 {
		return 0, 0
	}
	dismissal := time.Now().Add(4 * time.Hour).Unix()
	payloadBytes, err := BuildLiveActivityEndPayload(contentState, dismissal)
	if err != nil {
		slog.Error("failed to build live activity end payload", "error", err)
		return 0, 0
	}
	return s.sendToActivityTokens(ctx, tokens, payloadBytes)
}

// sendToActivityTokens sends raw payload to multiple activity tokens concurrently.
func (s *SenderService) sendToActivityTokens(ctx context.Context, tokens []models.ActivityToken, rawPayload []byte) (sent, failed int) {
	tokenStrs := make([]string, len(tokens))
	for i, at := range tokens {
		tokenStrs[i] = at.Token
	}
	return s.fanOutSend(ctx, tokenStrs,
		func(ctx context.Context, tok string) (*apns2.Response, error) {
			return s.client.SendLiveActivity(ctx, tok, rawPayload)
		},
		func(ctx context.Context, tok string) {
			_ = s.db.DeleteActivityToken(ctx, tok)
			slog.Info("deleted stale activity token (410)", "token_prefix", truncateToken(tok))
		},
	)
}

// sendFunc sends a push to a single token and returns the APNs response.
type sendFunc func(ctx context.Context, token string) (*apns2.Response, error)

// cleanupFunc handles 410 (Gone) token cleanup.
type cleanupFunc func(ctx context.Context, token string)

// fanOutSend sends pushes to multiple tokens concurrently with bounded parallelism.
// It calls sendFn for each token and cleanupFn on 410 responses.
func (s *SenderService) fanOutSend(ctx context.Context, tokens []string, sendFn sendFunc, cleanupFn cleanupFunc) (sent, failed int) {
	var mu sync.Mutex
	sem := make(chan struct{}, maxConcurrentPushes)
	var wg sync.WaitGroup

	for _, tok := range tokens {
		wg.Add(1)
		sem <- struct{}{} // acquire semaphore

		go func(token string) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore

			resp, err := sendFn(ctx, token)
			if err != nil {
				slog.Error("push send failed",
					"token_prefix", truncateToken(token),
					"error", err,
				)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			if resp.StatusCode == 410 {
				cleanupFn(ctx, token)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			if resp.StatusCode != 200 {
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			mu.Lock()
			sent++
			mu.Unlock()
		}(tok)
	}

	wg.Wait()
	return sent, failed
}
