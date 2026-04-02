package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hitl-sh/handoff-server/internal/models"
)

// UpsertUser creates or updates a user based on OAuth provider and ID.
func (db *DB) UpsertUser(ctx context.Context, user *models.User) error {
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO users (id, email, name, avatar_url, oauth_provider, oauth_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oauth_provider, oauth_id) DO UPDATE SET
			name = excluded.name,
			avatar_url = excluded.avatar_url,
			email = excluded.email,
			updated_at = excluded.updated_at
	`, user.ID, user.Email, user.Name, user.AvatarURL, user.OAuthProvider, user.OAuthID, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by their ID.
func (db *DB) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, email, name, avatar_url, oauth_provider, oauth_id, device_token, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.OAuthProvider, &u.OAuthID, &u.DeviceToken, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

// GetUserByOAuth retrieves a user by OAuth provider and ID.
func (db *DB) GetUserByOAuth(ctx context.Context, provider, oauthID string) (*models.User, error) {
	var u models.User
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, email, name, avatar_url, oauth_provider, oauth_id, device_token, created_at, updated_at
		FROM users WHERE oauth_provider = ? AND oauth_id = ?
	`, provider, oauthID).Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.OAuthProvider, &u.OAuthID, &u.DeviceToken, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by oauth: %w", err)
	}
	return &u, nil
}
