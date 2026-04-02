package db

import (
	"context"
	"fmt"

	"github.com/hitl-sh/handoff-server/internal/models"
)

// DashboardStats holds aggregate counts for the dashboard overview.
type DashboardStats struct {
	TotalLoops      int
	TotalRequests   int
	PendingRequests int
	CompletedToday  int
}

// GetDashboardStats returns aggregate statistics scoped to loops the user belongs to.
func (db *DB) GetDashboardStats(ctx context.Context, userID string) (DashboardStats, error) {
	var stats DashboardStats

	err := db.conn.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT lm.loop_id)
		FROM loop_members lm
		WHERE lm.user_id = ? AND lm.status = 'active'
	`, userID).Scan(&stats.TotalLoops)
	if err != nil {
		return stats, fmt.Errorf("count loops: %w", err)
	}

	err = db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM requests r
		JOIN loop_members lm ON r.loop_id = lm.loop_id AND lm.user_id = ? AND lm.status = 'active'
	`, userID).Scan(&stats.TotalRequests)
	if err != nil {
		return stats, fmt.Errorf("count requests: %w", err)
	}

	err = db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM requests r
		JOIN loop_members lm ON r.loop_id = lm.loop_id AND lm.user_id = ? AND lm.status = 'active'
		WHERE r.status = 'pending'
	`, userID).Scan(&stats.PendingRequests)
	if err != nil {
		return stats, fmt.Errorf("count pending: %w", err)
	}

	err = db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM requests r
		JOIN loop_members lm ON r.loop_id = lm.loop_id AND lm.user_id = ? AND lm.status = 'active'
		WHERE r.status = 'completed' AND r.response_at >= date('now')
	`, userID).Scan(&stats.CompletedToday)
	if err != nil {
		return stats, fmt.Errorf("count completed today: %w", err)
	}

	return stats, nil
}

// GetRecentRequests returns the most recent N requests across all loops the user belongs to.
func (db *DB) GetRecentRequests(ctx context.Context, userID string, limit int) ([]models.Request, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT r.id, r.loop_id, r.creator_id, r.api_key_id, r.agent_name, r.processing_type, r.content_type,
			r.priority, r.title, r.request_text, r.image_url, r.context_json, r.platform, r.response_type,
			r.response_config_json, r.default_response_json, r.timeout_seconds, r.timeout_at,
			r.callback_url, r.status, r.response_data_json, r.response_by, r.response_at,
			r.response_time_seconds, r.created_at, r.updated_at,
			COALESCE((SELECT l.name FROM loops l WHERE l.id = r.loop_id), '')
		FROM requests r
		JOIN loop_members lm ON r.loop_id = lm.loop_id AND lm.user_id = ? AND lm.status = 'active'
		ORDER BY r.created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent requests: %w", err)
	}
	defer rows.Close()

	var requests []models.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, *req)
	}
	return requests, nil
}

// GetLoopRequestCount returns the total number of requests in a loop.
func (db *DB) GetLoopRequestCount(ctx context.Context, loopID string) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM requests WHERE loop_id = ?
	`, loopID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get loop request count: %w", err)
	}
	return count, nil
}
