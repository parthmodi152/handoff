package api

import (
	"io/fs"
	"net/http"

	"github.com/hitl-sh/handoff-server/internal/config"
	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/events"
	"github.com/hitl-sh/handoff-server/internal/push"
)

// NewRouter creates and configures the HTTP router with all routes.
// pushSender, broker, and webhook may be nil (graceful degradation).
func NewRouter(database *db.DB, cfg *config.Config, pushSender push.Sender, broker *events.Broker, webhook *events.WebhookSender, webFS ...fs.FS) http.Handler {
	mux := http.NewServeMux()

	// --- Dashboard (web UI) ---
	if len(webFS) > 0 && webFS[0] != nil {
		staticFS, _ := fs.Sub(webFS[0], "static")
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

		dash := NewDashboardHandler(database, cfg, webFS[0])
		mux.HandleFunc("GET /{$}", dash.RootRedirect)
		mux.HandleFunc("GET /login", dash.LoginPage)
		mux.HandleFunc("GET /dashboard", dash.RequireSession(dash.DashboardPage))
		mux.HandleFunc("GET /loops", dash.RequireSession(dash.LoopsPage))
		mux.HandleFunc("GET /loops/new", dash.RequireSession(dash.LoopNewPage))
		mux.HandleFunc("POST /loops/new", dash.RequireSession(dash.LoopNewPage))
		mux.HandleFunc("GET /loops/{id}", dash.RequireSession(dash.LoopDetailPage))
		mux.HandleFunc("GET /api-keys", dash.RequireSession(dash.APIKeysPage))
		mux.HandleFunc("POST /api-keys", dash.RequireSession(dash.CreateAPIKey))
		mux.HandleFunc("POST /api-keys/{id}/revoke", dash.RequireSession(dash.RevokeAPIKey))
		mux.HandleFunc("GET /requests", dash.RequireSession(dash.RequestsPage))
		mux.HandleFunc("GET /requests/{id}", dash.RequireSession(dash.RequestDetailPage))
	}

	authMiddleware := NewAuthMiddleware(database, cfg)
	authHandler := NewAuthHandler(database, cfg)
	apiKeyHandler := NewAPIKeyHandler(database)
	loopHandler := NewLoopHandler(database)
	requestHandler := NewRequestHandler(database, pushSender, broker, webhook)
	deviceHandler := NewDeviceHandler(database)

	// --- Auth Endpoints (no auth required) ---
	mux.HandleFunc("GET /auth/google", authHandler.GoogleAuth)
	mux.HandleFunc("GET /auth/google/callback", authHandler.GoogleCallback)
	mux.HandleFunc("GET /auth/apple", authHandler.AppleAuth)
	mux.HandleFunc("GET /auth/apple/callback", authHandler.AppleCallback)
	mux.HandleFunc("POST /auth/apple/callback", authHandler.AppleCallback)
	mux.HandleFunc("POST /auth/apple/native", authHandler.AppleNativeAuth)

	// Dev login (only when HANDOFF_DEV_MODE=true)
	mux.HandleFunc("POST /auth/dev-login", authHandler.DevLogin)

	// Auth endpoints requiring auth
	mux.Handle("POST /auth/logout", authMiddleware.RequireAuth(http.HandlerFunc(authHandler.Logout)))
	mux.Handle("GET /auth/me", authMiddleware.RequireAuth(http.HandlerFunc(authHandler.Me)))

	// --- Test Endpoint ---
	mux.Handle("GET /api/v1/test", authMiddleware.RequireAuth(http.HandlerFunc(requestHandler.TestAPIKey)))

	// --- API Key Endpoints (session auth) ---
	mux.Handle("POST /api/v1/api-keys",
		authMiddleware.RequireSession(http.HandlerFunc(apiKeyHandler.CreateAPIKey)))
	mux.Handle("GET /api/v1/api-keys",
		authMiddleware.RequireAuth(http.HandlerFunc(apiKeyHandler.ListAPIKeys)))
	mux.Handle("DELETE /api/v1/api-keys/{id}",
		authMiddleware.RequireSession(http.HandlerFunc(apiKeyHandler.RevokeAPIKey)))

	// --- Loop Endpoints ---
	mux.Handle("POST /api/v1/loops",
		authMiddleware.RequireAuth(
			RequirePermission("loops:write")(http.HandlerFunc(loopHandler.CreateLoop))))
	mux.Handle("GET /api/v1/loops",
		authMiddleware.RequireAuth(
			RequirePermission("loops:read")(http.HandlerFunc(loopHandler.ListLoops))))
	mux.Handle("GET /api/v1/loops/{id}",
		authMiddleware.RequireAuth(
			RequirePermission("loops:read")(http.HandlerFunc(loopHandler.GetLoop))))

	// Join loop (session auth for mobile users)
	mux.Handle("POST /api/v1/loops/join",
		authMiddleware.RequireAuth(http.HandlerFunc(loopHandler.JoinLoop)))

	// --- Device Token Endpoints (session auth) ---
	mux.Handle("POST /api/v1/devices",
		authMiddleware.RequireSession(http.HandlerFunc(deviceHandler.RegisterDevice)))
	mux.Handle("DELETE /api/v1/devices/{token}",
		authMiddleware.RequireSession(http.HandlerFunc(deviceHandler.UnregisterDevice)))

	// --- Request Endpoints ---
	mux.Handle("POST /api/v1/loops/{loop_id}/requests",
		authMiddleware.RequireAuth(
			RequirePermission("requests:write")(http.HandlerFunc(requestHandler.CreateRequest))))
	mux.Handle("GET /api/v1/requests",
		authMiddleware.RequireAuth(
			RequirePermission("requests:read")(http.HandlerFunc(requestHandler.ListRequests))))
	mux.Handle("GET /api/v1/requests/{id}",
		authMiddleware.RequireAuth(
			RequirePermission("requests:read")(http.HandlerFunc(requestHandler.GetRequest))))
	mux.Handle("DELETE /api/v1/requests/{id}",
		authMiddleware.RequireAuth(
			RequirePermission("requests:write")(http.HandlerFunc(requestHandler.CancelRequest))))
	mux.Handle("POST /api/v1/requests/{id}/respond",
		authMiddleware.RequireAuth(http.HandlerFunc(requestHandler.RespondToRequest)))
	mux.Handle("GET /api/v1/requests/{id}/events",
		authMiddleware.RequireAuth(
			RequirePermission("requests:read")(http.HandlerFunc(requestHandler.StreamRequestEvents))))

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Apply global middleware chain: recovery -> request-id -> cors -> logging -> handler
	var handler http.Handler = mux
	handler = Logging(handler)
	handler = CORS(handler)
	handler = RequestID(handler)
	handler = Recovery(handler)

	return handler
}
