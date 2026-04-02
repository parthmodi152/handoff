package models

import "testing"

func TestHasLoopAccess(t *testing.T) {
	tests := []struct {
		name         string
		allowedLoops []string
		loopID       string
		want         bool
	}{
		{"empty allowed = unrestricted", nil, "any-loop", true},
		{"empty slice = unrestricted", []string{}, "any-loop", true},
		{"allowed loop", []string{"loop-a", "loop-b"}, "loop-a", true},
		{"second allowed loop", []string{"loop-a", "loop-b"}, "loop-b", true},
		{"disallowed loop", []string{"loop-a", "loop-b"}, "loop-c", false},
		{"single allowed", []string{"loop-a"}, "loop-a", true},
		{"single disallowed", []string{"loop-a"}, "loop-b", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &APIKey{AllowedLoops: tt.allowedLoops}
			if got := k.HasLoopAccess(tt.loopID); got != tt.want {
				t.Errorf("HasLoopAccess(%q) = %v, want %v", tt.loopID, got, tt.want)
			}
		})
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		perm        string
		want        bool
	}{
		{"has permission", []string{"loops:read", "requests:write"}, "loops:read", true},
		{"missing permission", []string{"loops:read"}, "requests:write", false},
		{"empty permissions", nil, "loops:read", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &APIKey{Permissions: tt.permissions}
			if got := k.HasPermission(tt.perm); got != tt.want {
				t.Errorf("HasPermission(%q) = %v, want %v", tt.perm, got, tt.want)
			}
		})
	}
}
