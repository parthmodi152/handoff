// Package push provides APNs push notification sending.
package push

import (
	"context"

	"github.com/sideshow/apns2/payload"
)

// Sender is the interface for sending push notifications.
// Implementations must be safe for concurrent use.
// A nil Sender means push is disabled — callers must nil-check before use.
type Sender interface {
	// NotifyLoopMembers sends a push to all devices of all members in a loop,
	// excluding the user identified by excludeUserID.
	NotifyLoopMembers(ctx context.Context, loopID, excludeUserID string, p *payload.Payload) (sent, failed int)

	// NotifyUser sends a push to all devices of a specific user.
	NotifyUser(ctx context.Context, userID string, p *payload.Payload) (sent, failed int)

	// UpdateLoopActivity sends a Live Activity update push to all activity tokens for a loop.
	UpdateLoopActivity(ctx context.Context, loopID string, contentState interface{}, staleDateUnix *int64, alertTitle, alertBody string) (sent, failed int)

	// EndLoopActivity sends a Live Activity end push to all activity tokens for a loop.
	EndLoopActivity(ctx context.Context, loopID string, contentState interface{}) (sent, failed int)
}
