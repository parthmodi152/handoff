package push

import (
	"context"
	"log/slog"

	"github.com/hitl-sh/handoff-server/internal/db"
)

// RefreshLoopActivity fetches the current pending state for a loop and sends
// the appropriate Live Activity push (update or end). It is safe to call from
// goroutines. alertTitle/alertBody are only used for "update" pushes (pass ""
// to send a silent update).
func RefreshLoopActivity(ctx context.Context, sender Sender, database *db.DB, loopID, alertTitle, alertBody string) {
	reqs, total, err := database.GetPendingRequestsByLoop(ctx, loopID, 5)
	if err != nil {
		slog.Error("failed to get pending for live activity", "loop_id", loopID, "error", err)
		return
	}
	cs := BuildLoopContentState(reqs, total)
	if total == 0 {
		sent, failed := sender.EndLoopActivity(ctx, loopID, cs)
		slog.Info("live activity ended", "loop_id", loopID, "sent", sent, "failed", failed)
	} else {
		stale := EarliestTimeoutUnix(reqs)
		sent, failed := sender.UpdateLoopActivity(ctx, loopID, cs, stale, alertTitle, alertBody)
		slog.Info("live activity updated", "loop_id", loopID, "sent", sent, "failed", failed)
	}
}
