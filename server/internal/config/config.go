package config

import (
	"os"
)

type Config struct {
	Port      string
	DBPath    string
	BaseURL   string
	JWTSecret string
	DevMode   bool

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string

	// Apple OAuth (web flow)
	AppleClientID     string
	AppleClientSecret string
	AppleTeamID       string
	AppleKeyID        string

	// Apple native iOS auth
	AppleBundleID string

	// APNs Push Notifications
	APNSKeyPath    string
	APNSKeyID      string
	APNSProduction bool
}

func Load() *Config {
	return &Config{
		Port:               getEnv("HANDOFF_PORT", "8080"),
		DBPath:             getEnv("HANDOFF_DB_PATH", "handoff.db"),
		BaseURL:            getEnv("HANDOFF_BASE_URL", "http://localhost:8080"),
		JWTSecret:          getEnv("HANDOFF_JWT_SECRET", "change-me-in-production-at-least-32-bytes!!"),
		DevMode:            getEnv("HANDOFF_DEV_MODE", "") == "true",
		GoogleClientID:     getEnv("HANDOFF_GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("HANDOFF_GOOGLE_CLIENT_SECRET", ""),
		AppleClientID:      getEnv("HANDOFF_APPLE_CLIENT_ID", ""),
		AppleClientSecret:  getEnv("HANDOFF_APPLE_CLIENT_SECRET", ""),
		AppleTeamID:        getEnv("HANDOFF_APPLE_TEAM_ID", ""),
		AppleKeyID:         getEnv("HANDOFF_APPLE_KEY_ID", ""),
		AppleBundleID:      getEnv("HANDOFF_APPLE_BUNDLE_ID", "sh.hitl.handoff"),
		APNSKeyPath:        getEnv("HANDOFF_APNS_KEY_PATH", ""),
		APNSKeyID:          getEnv("HANDOFF_APNS_KEY_ID", ""),
		APNSProduction:     getEnv("HANDOFF_APNS_PRODUCTION", "") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
