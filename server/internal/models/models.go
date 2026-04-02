package models

import (
	"encoding/json"
	"time"
)

// User represents an authenticated user.
type User struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	Name          string  `json:"name"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	OAuthProvider string  `json:"oauth_provider,omitempty"`
	OAuthID       string  `json:"-"`
	DeviceToken   *string `json:"-"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// APIKey represents a bearer token for programmatic API access.
type APIKey struct {
	ID           string   `json:"id"`
	UserID       string   `json:"user_id,omitempty"`
	Name         string   `json:"name"`
	KeyHash      string   `json:"-"`
	KeyPrefix    string   `json:"key_prefix"`
	Permissions  []string `json:"permissions"`
	AgentName    *string  `json:"agent_name,omitempty"`
	AllowedLoops []string `json:"allowed_loops,omitempty"`
	IsActive     bool     `json:"is_active"`
	LastUsedAt   *string  `json:"last_used_at,omitempty"`
	CreatedAt    string   `json:"created_at"`
	ExpiresAt    *string  `json:"expires_at,omitempty"`
}

// HasPermission checks if the API key has the given permission.
func (k *APIKey) HasPermission(perm string) bool {
	for _, p := range k.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// HasLoopAccess checks if the API key is allowed to access the given loop.
// Empty AllowedLoops means unrestricted access (backwards compatible).
func (k *APIKey) HasLoopAccess(loopID string) bool {
	if len(k.AllowedLoops) == 0 {
		return true
	}
	for _, id := range k.AllowedLoops {
		if id == loopID {
			return true
		}
	}
	return false
}

// Loop represents a named channel that groups review requests.
type Loop struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	CreatorID   string `json:"creator_id"`
	InviteCode  string `json:"invite_code"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// LoopWithMeta adds computed fields to a loop.
type LoopWithMeta struct {
	Loop
	InviteQR    string `json:"invite_qr,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
	Role        string `json:"role,omitempty"`
}

// LoopMember represents a user's membership in a loop.
type LoopMember struct {
	UserID    string  `json:"user_id"`
	Email     string  `json:"email,omitempty"`
	Name      string  `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
	JoinedAt  string  `json:"joined_at"`
}

// Request represents a single review task.
type Request struct {
	ID                  string           `json:"id"`
	LoopID              string           `json:"loop_id"`
	LoopName            string           `json:"loop_name,omitempty"`
	CreatorID           string           `json:"creator_id"`
	APIKeyID            *string          `json:"api_key_id,omitempty"`
	AgentName           *string          `json:"agent_name,omitempty"`
	ProcessingType      string           `json:"processing_type"`
	ContentType         string           `json:"content_type"`
	Priority            string           `json:"priority"`
	Title               string           `json:"title"`
	RequestText         string           `json:"request_text"`
	ImageURL            *string          `json:"image_url,omitempty"`
	Context             json.RawMessage  `json:"context,omitempty"`
	Platform            string           `json:"platform,omitempty"`
	ResponseType        string           `json:"response_type"`
	ResponseConfig      json.RawMessage  `json:"response_config"`
	DefaultResponse     json.RawMessage  `json:"default_response,omitempty"`
	TimeoutSeconds      *int             `json:"timeout_seconds,omitempty"`
	TimeoutAt           *string          `json:"timeout_at,omitempty"`
	CallbackURL         *string          `json:"callback_url,omitempty"`
	Status              string           `json:"status"`
	ResponseData        json.RawMessage  `json:"response_data,omitempty"`
	ResponseBy          *string          `json:"response_by,omitempty"`
	ResponseAt          *string          `json:"response_at,omitempty"`
	ResponseTimeSeconds *float64         `json:"response_time_seconds,omitempty"`
	CreatedAt           string           `json:"created_at"`
	UpdatedAt           string           `json:"updated_at"`
}

// DeviceToken represents a registered push notification device.
type DeviceToken struct {
	ID         string  `json:"id"`
	UserID     string  `json:"user_id"`
	Token      string  `json:"token"`
	Platform   string  `json:"platform"`
	AppVersion *string `json:"app_version,omitempty"`
	LastUsed   string  `json:"last_used"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

// ActivityToken represents a Live Activity push token registered for a loop.
type ActivityToken struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	LoopID    string `json:"loop_id"`
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateRequestInput is the input for creating a request.
type CreateRequestInput struct {
	ProcessingType  string          `json:"processing_type"`
	Type            string          `json:"type"`
	Priority        string          `json:"priority"`
	Title           string          `json:"title"`
	RequestText     string          `json:"request_text"`
	ImageURL        *string         `json:"image_url"`
	Context         json.RawMessage `json:"context"`
	ResponseType    string          `json:"response_type"`
	ResponseConfig  json.RawMessage `json:"response_config"`
	DefaultResponse json.RawMessage `json:"default_response"`
	TimeoutSeconds  *int            `json:"timeout_seconds"`
	CallbackURL     *string         `json:"callback_url"`
	Platform        string          `json:"platform"`
}

// RespondInput is the input for responding to a request.
type RespondInput struct {
	ResponseData json.RawMessage `json:"response_data"`
}

// NowUTC returns the current time formatted as ISO 8601 UTC.
func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
