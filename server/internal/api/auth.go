package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/hitl-sh/handoff-server/internal/config"
	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/models"
)

// AuthHandler handles OAuth authentication endpoints.
type AuthHandler struct {
	db  *db.DB
	cfg *config.Config
}

func NewAuthHandler(database *db.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: database, cfg: cfg}
}

func (h *AuthHandler) googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.cfg.GoogleClientID,
		ClientSecret: h.cfg.GoogleClientSecret,
		RedirectURL:  h.cfg.BaseURL + "/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func (h *AuthHandler) appleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.cfg.AppleClientID,
		ClientSecret: h.cfg.AppleClientSecret,
		RedirectURL:  h.cfg.BaseURL + "/auth/apple/callback",
		Scopes:       []string{"name", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://appleid.apple.com/auth/authorize",
			TokenURL: "https://appleid.apple.com/auth/token",
		},
	}
}

// GoogleAuth redirects to Google OAuth.
func (h *AuthHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")

	state := generateState()
	verifier := generateCodeVerifier()

	// Store state and verifier in cookies
	stateParts := state
	if platform == "mobile" {
		stateParts = state + "|mobile"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    stateParts,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_verifier",
		Value:    verifier,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	cfg := h.googleOAuthConfig()
	challenge := generateCodeChallenge(verifier)
	url := cfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("access_type", "offline"),
	)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallback handles the OAuth callback from Google.
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "Missing code or state parameter")
		return
	}

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing state cookie")
		return
	}

	isMobile := false
	storedState := stateCookie.Value
	if strings.HasSuffix(storedState, "|mobile") {
		isMobile = true
		storedState = strings.TrimSuffix(storedState, "|mobile")
	}
	if storedState != state {
		writeError(w, http.StatusBadRequest, "State mismatch")
		return
	}

	// Get verifier
	verifierCookie, err := r.Cookie("oauth_verifier")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing verifier cookie")
		return
	}

	// Exchange code for token
	cfg := h.googleOAuthConfig()
	token, err := cfg.Exchange(context.Background(), code,
		oauth2.SetAuthURLParam("code_verifier", verifierCookie.Value),
	)
	if err != nil {
		slog.Error("oauth token exchange failed", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to exchange authorization code")
		return
	}

	// Fetch user info from Google
	client := cfg.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch user info")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var googleUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(body, &googleUser); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse user info")
		return
	}

	// Upsert user
	now := models.NowUTC()
	var avatarURL *string
	if googleUser.Picture != "" {
		avatarURL = &googleUser.Picture
	}

	// Check if user exists
	existingUser, _ := h.db.GetUserByOAuth(r.Context(), "google", googleUser.ID)
	userID := uuid.New().String()
	if existingUser != nil {
		userID = existingUser.ID
	}

	user := &models.User{
		ID:            userID,
		Email:         googleUser.Email,
		Name:          googleUser.Name,
		AvatarURL:     avatarURL,
		OAuthProvider: "google",
		OAuthID:       googleUser.ID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := h.db.UpsertUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate JWT
	jwtToken := h.generateJWT(user)

	// Clear OAuth cookies
	clearCookie(w, "oauth_state")
	clearCookie(w, "oauth_verifier")

	if isMobile {
		// Redirect to custom URL scheme so WebAuthenticationSession can capture the token
		callbackURL := "handoff://auth/callback?token=" + url.QueryEscape(jwtToken)
		http.Redirect(w, r, callbackURL, http.StatusTemporaryRedirect)
		return
	}

	// Web flow: set cookie and redirect
	http.SetCookie(w, &http.Cookie{
		Name:     "handoff_session",
		Value:    jwtToken,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
}

// AppleAuth redirects to Apple OAuth.
func (h *AuthHandler) AppleAuth(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")

	state := generateState()
	stateParts := state
	if platform == "mobile" {
		stateParts = state + "|mobile"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    stateParts,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	cfg := h.appleOAuthConfig()
	url := cfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("response_mode", "form_post"),
	)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// AppleCallback handles the OAuth callback from Apple.
func (h *AuthHandler) AppleCallback(w http.ResponseWriter, r *http.Request) {
	// Apple sends callback as POST with form_post
	if r.Method == http.MethodPost {
		r.ParseForm()
	}

	code := r.FormValue("code")
	state := r.FormValue("state")
	if code == "" {
		code = r.URL.Query().Get("code")
	}
	if state == "" {
		state = r.URL.Query().Get("state")
	}

	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "Missing code or state parameter")
		return
	}

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing state cookie")
		return
	}

	isMobile := false
	storedState := stateCookie.Value
	if strings.HasSuffix(storedState, "|mobile") {
		isMobile = true
		storedState = strings.TrimSuffix(storedState, "|mobile")
	}
	if storedState != state {
		writeError(w, http.StatusBadRequest, "State mismatch")
		return
	}

	// Exchange code for tokens
	cfg := h.appleOAuthConfig()
	token, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		slog.Error("apple token exchange failed", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to exchange authorization code")
		return
	}

	// Extract email and sub from id_token
	idTokenStr, ok := token.Extra("id_token").(string)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Missing id_token from Apple")
		return
	}

	// Parse id_token without verification (Apple's token endpoint already validated)
	parts := strings.Split(idTokenStr, ".")
	if len(parts) < 2 {
		writeError(w, http.StatusInternalServerError, "Invalid id_token format")
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to decode id_token")
		return
	}
	var idClaims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(payload, &idClaims); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse id_token claims")
		return
	}

	// Apple only sends user name on first authorization
	userName := "Apple User"
	if userJSON := r.FormValue("user"); userJSON != "" {
		var appleUser struct {
			Name struct {
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
			} `json:"name"`
		}
		if json.Unmarshal([]byte(userJSON), &appleUser) == nil && appleUser.Name.FirstName != "" {
			userName = appleUser.Name.FirstName + " " + appleUser.Name.LastName
		}
	}

	// Upsert user
	now := models.NowUTC()
	existingUser, _ := h.db.GetUserByOAuth(r.Context(), "apple", idClaims.Sub)
	userID := uuid.New().String()
	if existingUser != nil {
		userID = existingUser.ID
		if existingUser.Name != "" {
			userName = existingUser.Name // Keep existing name
		}
	}

	user := &models.User{
		ID:            userID,
		Email:         idClaims.Email,
		Name:          userName,
		OAuthProvider: "apple",
		OAuthID:       idClaims.Sub,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := h.db.UpsertUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	jwtToken := h.generateJWT(user)
	clearCookie(w, "oauth_state")

	if isMobile {
		// Redirect to custom URL scheme so WebAuthenticationSession can capture the token
		callbackURL := "handoff://auth/callback?token=" + url.QueryEscape(jwtToken)
		http.Redirect(w, r, callbackURL, http.StatusTemporaryRedirect)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "handoff_session",
		Value:    jwtToken,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
}

// --- Apple JWKS verification for native Sign in with Apple ---

type appleJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type appleJWKSResponse struct {
	Keys []appleJWK `json:"keys"`
}

var (
	appleJWKSMu    sync.RWMutex
	appleJWKSKeys  []appleJWK
	appleJWKSFetch time.Time
)

func fetchAppleJWKS() ([]appleJWK, error) {
	appleJWKSMu.RLock()
	if time.Since(appleJWKSFetch) < 1*time.Hour && len(appleJWKSKeys) > 0 {
		keys := appleJWKSKeys
		appleJWKSMu.RUnlock()
		return keys, nil
	}
	appleJWKSMu.RUnlock()

	resp, err := http.Get("https://appleid.apple.com/auth/keys")
	if err != nil {
		return nil, fmt.Errorf("fetch apple JWKS: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read apple JWKS body: %w", err)
	}

	var jwks appleJWKSResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("parse apple JWKS: %w", err)
	}

	appleJWKSMu.Lock()
	appleJWKSKeys = jwks.Keys
	appleJWKSFetch = time.Now()
	appleJWKSMu.Unlock()

	return jwks.Keys, nil
}

func getApplePublicKey(kid string) (*rsa.PublicKey, error) {
	keys, err := fetchAppleJWKS()
	if err != nil {
		return nil, err
	}

	for _, k := range keys {
		if k.Kid == kid && k.Kty == "RSA" {
			// Decode modulus
			nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
			if err != nil {
				return nil, fmt.Errorf("decode modulus: %w", err)
			}
			// Decode exponent
			eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
			if err != nil {
				return nil, fmt.Errorf("decode exponent: %w", err)
			}
			e := new(big.Int).SetBytes(eBytes).Int64()
			return &rsa.PublicKey{
				N: new(big.Int).SetBytes(nBytes),
				E: int(e),
			}, nil
		}
	}
	return nil, fmt.Errorf("apple public key not found for kid: %s", kid)
}

// AppleNativeAuth handles native iOS Sign in with Apple.
// It receives the identity token from the client and verifies it against Apple's JWKS.
func (h *AuthHandler) AppleNativeAuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IdentityToken string `json:"identity_token"`
		Nonce         string `json:"nonce"`
		FullName      string `json:"full_name"`
		Email         string `json:"email"`
	}
	if !DecodeJSONBody(w, r, &req) {
		return
	}
	if req.IdentityToken == "" {
		writeError(w, http.StatusBadRequest, "Missing identity_token")
		return
	}

	// Parse the identity token header to get the kid
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(req.IdentityToken, jwt.MapClaims{})
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid identity token format")
		return
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		writeError(w, http.StatusBadRequest, "Missing kid in token header")
		return
	}

	// Get Apple's public key for this kid
	pubKey, err := getApplePublicKey(kid)
	if err != nil {
		slog.Error("failed to get apple public key", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to verify identity token")
		return
	}

	// Verify the token with Apple's public key
	claims := jwt.MapClaims{}
	verifiedToken, err := jwt.ParseWithClaims(req.IdentityToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil || !verifiedToken.Valid {
		slog.Error("apple identity token verification failed", "error", err)
		writeError(w, http.StatusUnauthorized, "Invalid identity token")
		return
	}

	// Verify issuer
	iss, _ := claims["iss"].(string)
	if iss != "https://appleid.apple.com" {
		writeError(w, http.StatusUnauthorized, "Invalid token issuer")
		return
	}

	// Verify audience (bundle ID)
	aud, _ := claims["aud"].(string)
	if aud != h.cfg.AppleBundleID {
		writeError(w, http.StatusUnauthorized, "Invalid token audience")
		return
	}

	// Verify nonce if provided
	if req.Nonce != "" {
		nonceHash := sha256.Sum256([]byte(req.Nonce))
		expectedNonce := hex.EncodeToString(nonceHash[:])
		tokenNonce, _ := claims["nonce"].(string)
		if tokenNonce != expectedNonce {
			writeError(w, http.StatusUnauthorized, "Nonce mismatch")
			return
		}
	}

	// Extract user info from verified claims
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	if sub == "" {
		writeError(w, http.StatusBadRequest, "Missing sub claim in identity token")
		return
	}

	// Use email from token if not provided in request
	if req.Email == "" {
		req.Email = email
	}

	// Upsert user
	now := models.NowUTC()
	existingUser, _ := h.db.GetUserByOAuth(r.Context(), "apple", sub)
	userID := uuid.New().String()
	userName := "Apple User"
	if existingUser != nil {
		userID = existingUser.ID
		if existingUser.Name != "" {
			userName = existingUser.Name
		}
	}
	// Apple only sends full name on first sign-in
	if req.FullName != "" {
		userName = req.FullName
	}

	user := &models.User{
		ID:            userID,
		Email:         req.Email,
		Name:          userName,
		OAuthProvider: "apple",
		OAuthID:       sub,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := h.db.UpsertUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	jwtToken := h.generateJWT(user)

	writeSuccess(w, http.StatusOK, "Authenticated", map[string]interface{}{
		"token": jwtToken,
		"user": map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"avatar_url": user.AvatarURL,
		},
	})
}

// Logout clears the session cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, "handoff_session")
	writeSuccess(w, http.StatusOK, "Logged out", nil)
}

// Me returns the currently authenticated user.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := RequireUser(w, r)
	if user == nil {
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"avatar_url":     user.AvatarURL,
			"oauth_provider": user.OAuthProvider,
			"created_at":     user.CreatedAt,
		},
	})
}

func (h *AuthHandler) generateJWT(user *models.User) string {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		slog.Error("failed to sign JWT", "error", err)
		return ""
	}
	return signed
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// DevLogin creates a test user and returns a JWT. Only available when HANDOFF_DEV_MODE=true.
func (h *AuthHandler) DevLogin(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.DevMode {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}

	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if !DecodeJSONBody(w, r, &req) {
		return
	}
	if req.Email == "" {
		req.Email = "dev@handoff.local"
	}
	if req.Name == "" {
		req.Name = "Dev User"
	}

	// Upsert user
	now := models.NowUTC()
	existingUser, _ := h.db.GetUserByOAuth(r.Context(), "dev", req.Email)
	userID := uuid.New().String()
	if existingUser != nil {
		userID = existingUser.ID
	}

	user := &models.User{
		ID:            userID,
		Email:         req.Email,
		Name:          req.Name,
		OAuthProvider: "dev",
		OAuthID:       req.Email,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := h.db.UpsertUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	jwtToken := h.generateJWT(user)

	http.SetCookie(w, &http.Cookie{
		Name:     "handoff_session",
		Value:    jwtToken,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	writeSuccess(w, http.StatusOK, "Dev login successful", map[string]interface{}{
		"token": jwtToken,
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
