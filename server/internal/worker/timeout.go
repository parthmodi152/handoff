// Package worker provides background workers for the Handoff server.
package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/events"
	"github.com/hitl-sh/handoff-server/internal/push"
)

// TimeoutWorker periodically checks for timed-out requests and updates their status.
type TimeoutWorker struct {
	db     *db.DB
	push   push.Sender    // may be nil
	broker *events.Broker // may be nil
	stop   chan struct{}
	done   chan struct{}
}

// NewTimeoutWorker creates a new timeout worker.
func NewTimeoutWorker(database *db.DB, pushSender push.Sender, broker *events.Broker) *TimeoutWorker {
	return &TimeoutWorker{
		db:     database,
		push:   pushSender,
		broker: broker,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// Start begins the timeout worker loop. It checks for timed-out requests at the given interval.
func (w *TimeoutWorker) Start(interval time.Duration) {
	slog.Info("timeout worker started", "interval", interval)

	go func() {
		defer close(w.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-w.stop:
				slog.Info("timeout worker stopped")
				return
			case <-ticker.C:
				w.tick()
			}
		}
	}()
}

// Stop signals the worker to stop and waits for it to finish.
func (w *TimeoutWorker) Stop() {
	close(w.stop)
	<-w.done
}

func (w *TimeoutWorker) tick() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in timeout worker", "recover", r)
		}
	}()

	ctx := context.Background()
	expired, err := w.db.ExpireTimedOutRequests(ctx)
	if err != nil {
		slog.Error("timeout worker: failed to expire requests", "error", err)
		return
	}

	if len(expired) == 0 {
		return
	}

	slog.Info("timeout worker: expired requests", "count", len(expired))

	// Send push notifications for each expired request
	if w.push != nil {
		for _, req := range expired {
			ntf := push.TimedOutPayload(req.Title, req.RequestText, req.LoopName, req.ID, req.LoopID)
			sent, failed := w.push.NotifyLoopMembers(ctx, req.LoopID, "", ntf)
			slog.Info("timeout push sent",
				"request_id", req.ID,
				"loop_id", req.LoopID,
				"sent", sent,
				"failed", failed,
			)
		}
	}

	// Update Live Activities for affected loops
	if w.push != nil {
		affectedLoops := make(map[string]bool)
		for _, req := range expired {
			affectedLoops[req.LoopID] = true
		}
		for loopID := range affectedLoops {
			push.RefreshLoopActivity(ctx, w.push, w.db, loopID, "", "")
		}
	}

	// Publish events to SSE subscribers
	if w.broker != nil {
		for _, req := range expired {
			eventData, _ := json.Marshal(map[string]interface{}{
				"request": map[string]interface{}{
					"id":     req.ID,
					"status": "timeout",
				},
			})
			w.broker.Publish(events.Event{
				RequestID: req.ID,
				LoopID:    req.LoopID,
				Status:    "timeout",
				Data:      eventData,
			})
		}
	}
}
