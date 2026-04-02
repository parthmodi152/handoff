package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hitl-sh/handoff-server/internal/models"
)

// CreateRequest inserts a new request.
func (db *DB) CreateRequest(ctx context.Context, req *models.Request) error {
	var contextJSON, defaultRespJSON, responseConfigJSON *string

	if req.Context != nil {
		s := string(req.Context)
		contextJSON = &s
	}
	if req.DefaultResponse != nil {
		s := string(req.DefaultResponse)
		defaultRespJSON = &s
	}
	if req.ResponseConfig != nil {
		s := string(req.ResponseConfig)
		responseConfigJSON = &s
	}

	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO requests (id, loop_id, creator_id, api_key_id, agent_name, processing_type, content_type, priority,
			title, request_text, image_url, context_json, platform, response_type, response_config_json,
			default_response_json, timeout_seconds, timeout_at, callback_url, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)
	`, req.ID, req.LoopID, req.CreatorID, req.APIKeyID, req.AgentName, req.ProcessingType, req.ContentType, req.Priority,
		req.Title, req.RequestText, req.ImageURL, contextJSON, req.Platform, req.ResponseType, responseConfigJSON,
		defaultRespJSON, req.TimeoutSeconds, req.TimeoutAt, req.CallbackURL, req.CreatedAt, req.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert request: %w", err)
	}
	return nil
}

// ListRequests returns requests accessible to the user with filters.
func (db *DB) ListRequests(ctx context.Context, userID string, status, priority, loopID, sort string, limit, offset int) ([]models.Request, int, error) {
	baseWhere := `
		FROM requests r
		JOIN loop_members lm ON r.loop_id = lm.loop_id AND lm.user_id = ? AND lm.status = 'active'
		WHERE 1=1
	`
	args := []interface{}{userID}

	if status != "" {
		baseWhere += " AND r.status = ?"
		args = append(args, status)
	}
	if priority != "" {
		baseWhere += " AND r.priority = ?"
		args = append(args, priority)
	}
	if loopID != "" {
		baseWhere += " AND r.loop_id = ?"
		args = append(args, loopID)
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseWhere
	if err := db.conn.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count requests: %w", err)
	}

	// Build order clause
	var orderClause string
	switch sort {
	case "created_at_asc":
		orderClause = "ORDER BY r.created_at ASC"
	case "priority_desc":
		orderClause = `ORDER BY CASE r.priority
			WHEN 'critical' THEN 0
			WHEN 'high' THEN 1
			WHEN 'medium' THEN 2
			WHEN 'low' THEN 3
			END ASC, r.created_at DESC`
	default:
		orderClause = "ORDER BY r.created_at DESC"
	}

	selectQuery := `
		SELECT r.id, r.loop_id, r.creator_id, r.api_key_id, r.agent_name, r.processing_type, r.content_type,
			r.priority, r.title, r.request_text, r.image_url, r.context_json, r.platform, r.response_type,
			r.response_config_json, r.default_response_json, r.timeout_seconds, r.timeout_at,
			r.callback_url, r.status, r.response_data_json, r.response_by, r.response_at,
			r.response_time_seconds, r.created_at, r.updated_at,
			COALESCE((SELECT l.name FROM loops l WHERE l.id = r.loop_id), '')
	` + baseWhere + " " + orderClause + " LIMIT ? OFFSET ?"

	args = append(args, limit, offset)

	rows, err := db.conn.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list requests: %w", err)
	}
	defer rows.Close()

	var requests []models.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, 0, err
		}
		requests = append(requests, *req)
	}
	return requests, total, nil
}

// GetRequest retrieves a single request by ID.
func (db *DB) GetRequest(ctx context.Context, id string) (*models.Request, error) {
	row := db.conn.QueryRowContext(ctx, `
		SELECT r.id, r.loop_id, r.creator_id, r.api_key_id, r.agent_name, r.processing_type, r.content_type,
			r.priority, r.title, r.request_text, r.image_url, r.context_json, r.platform, r.response_type,
			r.response_config_json, r.default_response_json, r.timeout_seconds, r.timeout_at,
			r.callback_url, r.status, r.response_data_json, r.response_by, r.response_at,
			r.response_time_seconds, r.created_at, r.updated_at,
			COALESCE((SELECT l.name FROM loops l WHERE l.id = r.loop_id), '')
		FROM requests r WHERE r.id = ?
	`, id)

	var req models.Request
	var contextJSON, responseConfigJSON, defaultRespJSON, responseDataJSON sql.NullString
	var loopName string
	err := row.Scan(&req.ID, &req.LoopID, &req.CreatorID, &req.APIKeyID, &req.AgentName, &req.ProcessingType, &req.ContentType,
		&req.Priority, &req.Title, &req.RequestText, &req.ImageURL, &contextJSON, &req.Platform, &req.ResponseType,
		&responseConfigJSON, &defaultRespJSON, &req.TimeoutSeconds, &req.TimeoutAt,
		&req.CallbackURL, &req.Status, &responseDataJSON, &req.ResponseBy, &req.ResponseAt,
		&req.ResponseTimeSeconds, &req.CreatedAt, &req.UpdatedAt, &loopName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}

	req.LoopName = loopName
	if contextJSON.Valid {
		req.Context = json.RawMessage(contextJSON.String)
	}
	if responseConfigJSON.Valid {
		req.ResponseConfig = json.RawMessage(responseConfigJSON.String)
	}
	if defaultRespJSON.Valid {
		req.DefaultResponse = json.RawMessage(defaultRespJSON.String)
	}
	if responseDataJSON.Valid {
		req.ResponseData = json.RawMessage(responseDataJSON.String)
	}

	return &req, nil
}

// CancelRequest sets a request status to cancelled.
func (db *DB) CancelRequest(ctx context.Context, id string) error {
	_, err := db.conn.ExecContext(ctx, `
		UPDATE requests SET status = 'cancelled', updated_at = datetime('now') WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("cancel request: %w", err)
	}
	return nil
}

// RespondToRequest submits a response to a request.
func (db *DB) RespondToRequest(ctx context.Context, reqID, userID string, responseData json.RawMessage) error {
	now := models.NowUTC()
	_, err := db.conn.ExecContext(ctx, `
		UPDATE requests SET
			status = 'completed',
			response_data_json = ?,
			response_by = ?,
			response_at = ?,
			response_time_seconds = (julianday(?) - julianday(created_at)) * 86400.0,
			updated_at = ?
		WHERE id = ? AND status = 'pending'
	`, string(responseData), userID, now, now, now, reqID)
	if err != nil {
		return fmt.Errorf("respond to request: %w", err)
	}
	return nil
}

// ExpireTimedOutRequests finds pending requests past their timeout and marks them as 'timeout'.
// Returns the expired requests so callers can send push notifications.
func (db *DB) ExpireTimedOutRequests(ctx context.Context) ([]models.Request, error) {
	now := models.NowUTC()

	// Find pending requests that have timed out
	rows, err := db.conn.QueryContext(ctx, `
		SELECT r.id, r.loop_id, r.creator_id, r.api_key_id, r.agent_name, r.processing_type, r.content_type,
			r.priority, r.title, r.request_text, r.image_url, r.context_json, r.platform, r.response_type,
			r.response_config_json, r.default_response_json, r.timeout_seconds, r.timeout_at,
			r.callback_url, r.status, r.response_data_json, r.response_by, r.response_at,
			r.response_time_seconds, r.created_at, r.updated_at,
			COALESCE(l.name, '') as loop_name
		FROM requests r
		LEFT JOIN loops l ON r.loop_id = l.id
		WHERE r.status = 'pending' AND r.timeout_at IS NOT NULL AND r.timeout_at < ?
	`, now)
	if err != nil {
		return nil, fmt.Errorf("query timed out requests: %w", err)
	}
	defer rows.Close()

	var expired []models.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		expired = append(expired, *req)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate timed out requests: %w", err)
	}

	// Update status to 'timeout'
	for _, req := range expired {
		_, err := db.conn.ExecContext(ctx, `
			UPDATE requests SET status = 'timeout', updated_at = ? WHERE id = ? AND status = 'pending'
		`, now, req.ID)
		if err != nil {
			return nil, fmt.Errorf("update request %s to timeout: %w", req.ID, err)
		}
	}

	return expired, nil
}

// GetActiveLoopMemberCount returns the number of active members in a loop.
func (db *DB) GetActiveLoopMemberCount(ctx context.Context, loopID string) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM loop_members WHERE loop_id = ? AND status = 'active'
	`, loopID).Scan(&count)
	return count, err
}

// GetPendingRequestsByLoop returns pending requests for a loop, sorted by priority then timeout.
func (db *DB) GetPendingRequestsByLoop(ctx context.Context, loopID string, limit int) ([]models.Request, int, error) {
	var total int
	err := db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM requests WHERE loop_id = ? AND status = 'pending'
	`, loopID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count pending requests: %w", err)
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT r.id, r.loop_id, r.creator_id, r.api_key_id, r.agent_name, r.processing_type, r.content_type,
			r.priority, r.title, r.request_text, r.image_url, r.context_json, r.platform, r.response_type,
			r.response_config_json, r.default_response_json, r.timeout_seconds, r.timeout_at,
			r.callback_url, r.status, r.response_data_json, r.response_by, r.response_at,
			r.response_time_seconds, r.created_at, r.updated_at,
			COALESCE((SELECT l.name FROM loops l WHERE l.id = r.loop_id), '')
		FROM requests r
		WHERE r.loop_id = ? AND r.status = 'pending'
		ORDER BY CASE r.priority
			WHEN 'critical' THEN 0
			WHEN 'high' THEN 1
			WHEN 'medium' THEN 2
			WHEN 'low' THEN 3
			END ASC,
			r.timeout_at ASC NULLS LAST
		LIMIT ?
	`, loopID, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("get pending requests by loop: %w", err)
	}
	defer rows.Close()

	var requests []models.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, 0, err
		}
		requests = append(requests, *req)
	}
	return requests, total, rows.Err()
}

func scanRequest(rows *sql.Rows) (*models.Request, error) {
	var req models.Request
	var contextJSON, responseConfigJSON, defaultRespJSON, responseDataJSON sql.NullString
	var loopName string

	err := rows.Scan(&req.ID, &req.LoopID, &req.CreatorID, &req.APIKeyID, &req.AgentName, &req.ProcessingType, &req.ContentType,
		&req.Priority, &req.Title, &req.RequestText, &req.ImageURL, &contextJSON, &req.Platform, &req.ResponseType,
		&responseConfigJSON, &defaultRespJSON, &req.TimeoutSeconds, &req.TimeoutAt,
		&req.CallbackURL, &req.Status, &responseDataJSON, &req.ResponseBy, &req.ResponseAt,
		&req.ResponseTimeSeconds, &req.CreatedAt, &req.UpdatedAt, &loopName)
	if err != nil {
		return nil, fmt.Errorf("scan request: %w", err)
	}

	req.LoopName = loopName
	if contextJSON.Valid && contextJSON.String != "" {
		req.Context = json.RawMessage(contextJSON.String)
	}
	if responseConfigJSON.Valid && responseConfigJSON.String != "" {
		req.ResponseConfig = json.RawMessage(responseConfigJSON.String)
	}
	if defaultRespJSON.Valid && defaultRespJSON.String != "" {
		req.DefaultResponse = json.RawMessage(defaultRespJSON.String)
	}
	if responseDataJSON.Valid && responseDataJSON.String != "" {
		req.ResponseData = json.RawMessage(responseDataJSON.String)
	}

	// Check for timeout on read
	if req.Status == "pending" && req.TimeoutAt != nil {
		if *req.TimeoutAt <= models.NowUTC() {
			req.Status = "timeout"
			if req.DefaultResponse != nil {
				req.ResponseData = req.DefaultResponse
			}
		}
	}

	_ = strings.TrimSpace(loopName)
	return &req, nil
}
