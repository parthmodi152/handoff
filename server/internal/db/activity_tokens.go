package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hitl-sh/handoff-server/internal/models"
)

func (db *DB) UpsertActivityToken(ctx context.Context, userID, loopID, token string) error {
	now := models.NowUTC()
	id := uuid.New().String()
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO activity_tokens (id, user_id, loop_id, token, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET
			user_id = excluded.user_id,
			loop_id = excluded.loop_id,
			updated_at = excluded.updated_at
	`, id, userID, loopID, token, now, now)
	if err != nil {
		return fmt.Errorf("upsert activity token: %w", err)
	}
	return nil
}

func (db *DB) DeleteActivityToken(ctx context.Context, token string) error {
	_, err := db.conn.ExecContext(ctx, `DELETE FROM activity_tokens WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("delete activity token: %w", err)
	}
	return nil
}

func (db *DB) GetActivityTokensByLoop(ctx context.Context, loopID string) ([]models.ActivityToken, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, user_id, loop_id, token, created_at, updated_at
		FROM activity_tokens WHERE loop_id = ?
	`, loopID)
	if err != nil {
		return nil, fmt.Errorf("get activity tokens by loop: %w", err)
	}
	defer rows.Close()

	var tokens []models.ActivityToken
	for rows.Next() {
		var t models.ActivityToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.LoopID, &t.Token, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan activity token: %w", err)
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (db *DB) DeleteActivityTokensByUser(ctx context.Context, userID string) error {
	_, err := db.conn.ExecContext(ctx, `DELETE FROM activity_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete activity tokens by user: %w", err)
	}
	return nil
}
