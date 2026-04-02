package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/hitl-sh/handoff-server/internal/config"
	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/models"
)

type contextKey string

const (
	ContextKeyUser       contextKey = "user"
	ContextKeyAPIKey     contextKey = "apikey"
	ContextKeyAuthMethod contextKey = "auth_method"
	ContextKeyRequestID  contextKey = "request_id"
)

// Recovery middleware catches panics and returns 500.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "error", err, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestID adds a unique request ID to each request.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		ctx := context.WithValue(r.Context(), ContextKeyRequestID, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logging logs each request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", time.Since(start).String(),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush passes through to the underlying writer if it supports http.Flusher (needed for SSE).
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter (needed for http.NewResponseController).
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// CORS adds CORS headers.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware handles both JWT session auth and API key auth.
type AuthMiddleware struct {
	db  *db.DB
	cfg *config.Config
}

func NewAuthMiddleware(database *db.DB, cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{db: database, cfg: cfg}
}

// RequireAuth requires authentication via JWT or API key.
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				writeError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			// Distinguish JWT from API key
			if strings.HasPrefix(token, "hnd_") {
				am.authenticateAPIKey(w, r, next, token)
				return
			}
			if strings.HasPrefix(token, "eyJ") {
				am.authenticateJWT(w, r, next, token)
				return
			}
			writeError(w, http.StatusUnauthorized, "Invalid token format")
			return
		}

		// Check cookie
		cookie, err := r.Cookie("handoff_session")
		if err == nil && cookie.Value != "" {
			am.authenticateJWT(w, r, next, cookie.Value)
			return
		}

		writeError(w, http.StatusUnauthorized, "Authentication required")
	})
}

// RequireSession requires JWT session auth only (no API key).
func (am *AuthMiddleware) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Authorization header for JWT
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != authHeader && strings.HasPrefix(token, "eyJ") {
				am.authenticateJWT(w, r, next, token)
				return
			}
		}

		// Check cookie
		cookie, err := r.Cookie("handoff_session")
		if err == nil && cookie.Value != "" {
			am.authenticateJWT(w, r, next, cookie.Value)
			return
		}

		writeError(w, http.StatusUnauthorized, "Session authentication required")
	})
}

func (am *AuthMiddleware) authenticateJWT(w http.ResponseWriter, r *http.Request, next http.Handler, tokenString string) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(am.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		writeError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Invalid token claims")
		return
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Invalid token: missing sub")
		return
	}

	user, err := am.db.GetUserByID(r.Context(), userID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "User not found")
		return
	}

	ctx := context.WithValue(r.Context(), ContextKeyUser, user)
	ctx = context.WithValue(ctx, ContextKeyAuthMethod, "session")
	next.ServeHTTP(w, r.WithContext(ctx))
}

func (am *AuthMiddleware) authenticateAPIKey(w http.ResponseWriter, r *http.Request, next http.Handler, rawKey string) {
	if len(rawKey) < 8 {
		writeError(w, http.StatusUnauthorized, "Invalid API key")
		return
	}

	prefix := rawKey[:8]
	keys, err := am.db.GetAPIKeysByPrefix(r.Context(), prefix)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Authentication error")
		return
	}

	var matchedKey *models.APIKey
	for _, k := range keys {
		if err := bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(rawKey)); err == nil {
			kCopy := k
			matchedKey = &kCopy
			break
		}
	}

	if matchedKey == nil {
		writeError(w, http.StatusUnauthorized, "Invalid API key")
		return
	}

	user, err := am.db.GetUserByID(r.Context(), matchedKey.UserID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "API key owner not found")
		return
	}

	// Update last_used_at async
	go am.db.UpdateAPIKeyLastUsed(context.Background(), matchedKey.ID)

	ctx := context.WithValue(r.Context(), ContextKeyUser, user)
	ctx = context.WithValue(ctx, ContextKeyAPIKey, matchedKey)
	ctx = context.WithValue(ctx, ContextKeyAuthMethod, "apikey")
	next.ServeHTTP(w, r.WithContext(ctx))
}

// RequirePermission checks that the API key has the required permission.
// Session auth (no API key) has full access.
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey, _ := r.Context().Value(ContextKeyAPIKey).(*models.APIKey)
			if apiKey == nil {
				// Session auth — full access
				next.ServeHTTP(w, r)
				return
			}
			if !apiKey.HasPermission(permission) {
				writeError(w, http.StatusForbidden, "Insufficient permissions: requires "+permission)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CheckAPIKeyLoopAccess returns an error if an API key is in use and lacks access to the loop.
// Session auth (no API key) is always allowed.
func CheckAPIKeyLoopAccess(r *http.Request, loopID string) error {
	apiKey := GetAPIKeyFromContext(r)
	if apiKey == nil {
		return nil
	}
	if !apiKey.HasLoopAccess(loopID) {
		return fmt.Errorf("API key does not have access to this loop")
	}
	return nil
}

// EnforceLoopAccess checks API key loop access and writes a 403 if denied.
// Returns true if access is granted, false if denied (response already written).
func EnforceLoopAccess(w http.ResponseWriter, r *http.Request, loopID string) bool {
	if err := CheckAPIKeyLoopAccess(r, loopID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return false
	}
	return true
}

// GetUserFromContext retrieves the authenticated user from context.
func GetUserFromContext(r *http.Request) *models.User {
	user, _ := r.Context().Value(ContextKeyUser).(*models.User)
	return user
}

// GetAPIKeyFromContext retrieves the API key from context (nil for session auth).
func GetAPIKeyFromContext(r *http.Request) *models.APIKey {
	key, _ := r.Context().Value(ContextKeyAPIKey).(*models.APIKey)
	return key
}
