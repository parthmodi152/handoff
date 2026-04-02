package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hitl-sh/handoff-server/internal/models"
)

// CreateLoop inserts a new loop and adds the creator as a member.
func (db *DB) CreateLoop(ctx context.Context, loop *models.Loop) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO loops (id, name, description, icon, creator_id, invite_code, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, loop.ID, loop.Name, loop.Description, loop.Icon, loop.CreatorID, loop.InviteCode, loop.CreatedAt, loop.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert loop: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO loop_members (loop_id, user_id, role, status, joined_at)
		VALUES (?, ?, 'creator', 'active', ?)
	`, loop.ID, loop.CreatorID, loop.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert loop member: %w", err)
	}

	return tx.Commit()
}

// ListLoops returns all loops where the user is an active member.
func (db *DB) ListLoops(ctx context.Context, userID string) ([]models.LoopWithMeta, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT l.id, l.name, l.description, l.icon, l.creator_id, l.invite_code, l.created_at, l.updated_at,
			lm.role,
			(SELECT COUNT(*) FROM loop_members WHERE loop_id = l.id AND status = 'active') as member_count
		FROM loops l
		JOIN loop_members lm ON l.id = lm.loop_id AND lm.user_id = ?
		WHERE lm.status = 'active'
		ORDER BY l.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list loops: %w", err)
	}
	defer rows.Close()

	var loops []models.LoopWithMeta
	for rows.Next() {
		var lm models.LoopWithMeta
		if err := rows.Scan(&lm.ID, &lm.Name, &lm.Description, &lm.Icon, &lm.CreatorID,
			&lm.InviteCode, &lm.CreatedAt, &lm.UpdatedAt, &lm.Role, &lm.MemberCount); err != nil {
			return nil, fmt.Errorf("scan loop: %w", err)
		}
		loops = append(loops, lm)
	}
	return loops, nil
}

// GetLoop retrieves a loop by ID.
func (db *DB) GetLoop(ctx context.Context, id string) (*models.Loop, error) {
	var l models.Loop
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, name, description, icon, creator_id, invite_code, created_at, updated_at
		FROM loops WHERE id = ?
	`, id).Scan(&l.ID, &l.Name, &l.Description, &l.Icon, &l.CreatorID, &l.InviteCode, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get loop: %w", err)
	}
	return &l, nil
}

// GetLoopByInviteCode retrieves a loop by its invite code.
func (db *DB) GetLoopByInviteCode(ctx context.Context, code string) (*models.Loop, error) {
	var l models.Loop
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, name, description, icon, creator_id, invite_code, created_at, updated_at
		FROM loops WHERE UPPER(invite_code) = UPPER(?)
	`, code).Scan(&l.ID, &l.Name, &l.Description, &l.Icon, &l.CreatorID, &l.InviteCode, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get loop by invite code: %w", err)
	}
	return &l, nil
}

// IsLoopMember checks if a user is an active member of a loop.
func (db *DB) IsLoopMember(ctx context.Context, loopID, userID string) (bool, string, error) {
	var role string
	err := db.conn.QueryRowContext(ctx, `
		SELECT role FROM loop_members WHERE loop_id = ? AND user_id = ? AND status = 'active'
	`, loopID, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("is loop member: %w", err)
	}
	return true, role, nil
}

// GetLoopMembers returns all members of a loop.
func (db *DB) GetLoopMembers(ctx context.Context, loopID string) ([]models.LoopMember, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT lm.user_id, u.email, u.name, u.avatar_url, lm.role, lm.status, lm.joined_at
		FROM loop_members lm
		JOIN users u ON lm.user_id = u.id
		WHERE lm.loop_id = ?
		ORDER BY lm.joined_at ASC
	`, loopID)
	if err != nil {
		return nil, fmt.Errorf("get loop members: %w", err)
	}
	defer rows.Close()

	var members []models.LoopMember
	for rows.Next() {
		var m models.LoopMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.Name, &m.AvatarURL, &m.Role, &m.Status, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan loop member: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

// JoinLoop adds a user to a loop.
func (db *DB) JoinLoop(ctx context.Context, loopID, userID string) error {
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO loop_members (loop_id, user_id, role, status, joined_at)
		VALUES (?, ?, 'member', 'active', datetime('now'))
	`, loopID, userID)
	if err != nil {
		return fmt.Errorf("join loop: %w", err)
	}
	return nil
}

// GetLoopMemberCount returns the number of active members in a loop.
func (db *DB) GetLoopMemberCount(ctx context.Context, loopID string) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM loop_members WHERE loop_id = ? AND status = 'active'
	`, loopID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get loop member count: %w", err)
	}
	return count, nil
}

// InviteCodeExists checks if an invite code is already in use.
func (db *DB) InviteCodeExists(ctx context.Context, code string) (bool, error) {
	var count int
	err := db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM loops WHERE invite_code = ?
	`, code).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("invite code exists: %w", err)
	}
	return count > 0, nil
}
