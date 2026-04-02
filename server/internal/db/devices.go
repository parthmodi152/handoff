package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/hitl-sh/handoff-server/internal/models"
)

// UpsertDeviceToken inserts or updates a device token.
// If the token already exists (e.g., device sold or account switch), it reassigns to the new user.
func (db *DB) UpsertDeviceToken(ctx context.Context, userID, token, platform, appVersion string) error {
	now := models.NowUTC()
	id := uuid.New().String()

	var appVersionPtr *string
	if appVersion != "" {
		appVersionPtr = &appVersion
	}

	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO device_tokens (id, user_id, token, platform, app_version, last_used, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET
			user_id = excluded.user_id,
			platform = excluded.platform,
			app_version = excluded.app_version,
			last_used = excluded.last_used,
			updated_at = excluded.updated_at
	`, id, userID, token, platform, appVersionPtr, now, now, now)
	if err != nil {
		return fmt.Errorf("upsert device token: %w", err)
	}
	return nil
}

// DeleteDeviceToken removes a device token by its token string.
func (db *DB) DeleteDeviceToken(ctx context.Context, token string) error {
	_, err := db.conn.ExecContext(ctx, `DELETE FROM device_tokens WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("delete device token: %w", err)
	}
	return nil
}

// GetDeviceTokensByUser returns all device tokens for a user.
func (db *DB) GetDeviceTokensByUser(ctx context.Context, userID string) ([]models.DeviceToken, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, user_id, token, platform, app_version, last_used, created_at, updated_at
		FROM device_tokens
		WHERE user_id = ?
		ORDER BY last_used DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get device tokens by user: %w", err)
	}
	defer rows.Close()

	return scanDeviceTokens(rows)
}

// GetDeviceTokensByLoopMembers returns all device tokens for members of a loop,
// optionally excluding a specific user.
func (db *DB) GetDeviceTokensByLoopMembers(ctx context.Context, loopID, excludeUserID string) ([]models.DeviceToken, error) {
	query := `
		SELECT dt.id, dt.user_id, dt.token, dt.platform, dt.app_version, dt.last_used, dt.created_at, dt.updated_at
		FROM device_tokens dt
		JOIN loop_members lm ON dt.user_id = lm.user_id
		WHERE lm.loop_id = ? AND lm.status = 'active'
	`
	args := []interface{}{loopID}

	if excludeUserID != "" {
		query += ` AND dt.user_id != ?`
		args = append(args, excludeUserID)
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get device tokens by loop members: %w", err)
	}
	defer rows.Close()

	return scanDeviceTokens(rows)
}

// GetPendingCountForUser returns the number of pending requests across all loops the user is a member of.
func (db *DB) GetPendingCountForUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM requests r
		JOIN loop_members lm ON r.loop_id = lm.loop_id
		WHERE lm.user_id = ? AND lm.status = 'active' AND r.status = 'pending'
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get pending count for user: %w", err)
	}
	return count, nil
}

// DeleteDeviceTokensByUser removes all device tokens for a user (used on sign-out).
func (db *DB) DeleteDeviceTokensByUser(ctx context.Context, userID string) error {
	_, err := db.conn.ExecContext(ctx, `DELETE FROM device_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete device tokens by user: %w", err)
	}
	return nil
}

func scanDeviceTokens(rows *sql.Rows) ([]models.DeviceToken, error) {
	var tokens []models.DeviceToken
	for rows.Next() {
		var dt models.DeviceToken
		var appVersion sql.NullString
		if err := rows.Scan(&dt.ID, &dt.UserID, &dt.Token, &dt.Platform, &appVersion, &dt.LastUsed, &dt.CreatedAt, &dt.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan device token: %w", err)
		}
		if appVersion.Valid {
			dt.AppVersion = &appVersion.String
		}
		tokens = append(tokens, dt)
	}
	return tokens, rows.Err()
}
