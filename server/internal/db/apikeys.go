package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hitl-sh/handoff-server/internal/models"
)

// CreateAPIKey inserts a new API key record and its loop scope entries.
func (db *DB) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	permsJSON, err := json.Marshal(key.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, permissions, agent_name, is_active, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`, key.ID, key.UserID, key.Name, key.KeyHash, key.KeyPrefix, string(permsJSON), key.AgentName, key.CreatedAt, key.ExpiresAt)
	if err != nil {
		return fmt.Errorf("insert api key: %w", err)
	}

	for _, loopID := range key.AllowedLoops {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO api_key_loops (api_key_id, loop_id) VALUES (?, ?)
		`, key.ID, loopID)
		if err != nil {
			return fmt.Errorf("insert api_key_loop: %w", err)
		}
	}

	return tx.Commit()
}

// ListAPIKeys returns all API keys for a user.
func (db *DB) ListAPIKeys(ctx context.Context, userID string) ([]models.APIKey, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, user_id, name, key_prefix, permissions, agent_name, is_active, last_used_at, created_at, expires_at
		FROM api_keys WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var k models.APIKey
		var permsJSON string
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &permsJSON, &k.AgentName, &k.IsActive, &k.LastUsedAt, &k.CreatedAt, &k.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		json.Unmarshal([]byte(permsJSON), &k.Permissions)
		keys = append(keys, k)
	}

	if err := loadAllowedLoops(ctx, db, keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// GetAPIKeysByPrefix returns active API keys matching the given prefix.
func (db *DB) GetAPIKeysByPrefix(ctx context.Context, prefix string) ([]models.APIKey, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, user_id, name, key_hash, key_prefix, permissions, agent_name, is_active, last_used_at, created_at, expires_at
		FROM api_keys
		WHERE key_prefix = ? AND is_active = 1 AND (expires_at IS NULL OR expires_at > datetime('now'))
	`, prefix)
	if err != nil {
		return nil, fmt.Errorf("get api keys by prefix: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var k models.APIKey
		var permsJSON string
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.KeyPrefix, &permsJSON, &k.AgentName, &k.IsActive, &k.LastUsedAt, &k.CreatedAt, &k.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		json.Unmarshal([]byte(permsJSON), &k.Permissions)
		keys = append(keys, k)
	}

	if err := loadAllowedLoops(ctx, db, keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// RevokeAPIKey sets an API key as inactive.
func (db *DB) RevokeAPIKey(ctx context.Context, id, userID string) error {
	res, err := db.conn.ExecContext(ctx, `
		UPDATE api_keys SET is_active = 0 WHERE id = ? AND user_id = ?
	`, id, userID)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

// UpdateAPIKeyLastUsed updates the last_used_at timestamp.
func (db *DB) UpdateAPIKeyLastUsed(ctx context.Context, id string) {
	db.conn.ExecContext(ctx, `UPDATE api_keys SET last_used_at = datetime('now') WHERE id = ?`, id)
}

// loadAllowedLoops populates AllowedLoops for each key from the api_key_loops table.
func loadAllowedLoops(ctx context.Context, database *DB, keys []models.APIKey) error {
	for i := range keys {
		rows, err := database.conn.QueryContext(ctx, `
			SELECT loop_id FROM api_key_loops WHERE api_key_id = ?
		`, keys[i].ID)
		if err != nil {
			return fmt.Errorf("load allowed loops: %w", err)
		}
		for rows.Next() {
			var loopID string
			if err := rows.Scan(&loopID); err != nil {
				rows.Close()
				return fmt.Errorf("scan allowed loop: %w", err)
			}
			keys[i].AllowedLoops = append(keys[i].AllowedLoops, loopID)
		}
		rows.Close()
	}
	return nil
}
