package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/hitl-sh/handoff-server/internal/config"
	"github.com/hitl-sh/handoff-server/internal/db"
	"github.com/hitl-sh/handoff-server/internal/models"
)

// DashboardHandler serves the web dashboard HTML pages.
type DashboardHandler struct {
	db        *db.DB
	cfg       *config.Config
	templates map[string]*template.Template
}

// PageData is the base data passed to every dashboard template.
type PageData struct {
	ActiveNav string
	User      *models.User
	Flash     string
}

// DashboardPageData is the data for the main dashboard overview page.
type DashboardPageData struct {
	PageData
	Stats          db.DashboardStats
	RecentRequests []models.Request
}

// LoopsPageData is the data for the loops listing page.
type LoopsPageData struct {
	PageData
	Loops []models.LoopWithMeta
}

// LoopDetailPageData is the data for a single loop's detail page.
type LoopDetailPageData struct {
	PageData
	Loop     *models.Loop
	InviteQR string
	Members  []models.LoopMember
	Requests []models.Request
}

// APIKeysPageData is the data for the API keys management page.
type APIKeysPageData struct {
	PageData
	Keys   []models.APIKey
	Loops  []models.LoopWithMeta
	NewKey string
}

// RequestsPageData is the data for the requests listing page.
type RequestsPageData struct {
	PageData
	Requests   []models.Request
	Loops      []models.LoopWithMeta
	Status     string
	Priority   string
	LoopID     string
	Page       int
	TotalPages int
	Total      int
}

// RequestDetailPageData is the data for a single request's detail page.
type RequestDetailPageData struct {
	PageData
	Request *models.Request
}

// NewDashboardHandler creates a DashboardHandler and parses all templates from the embedded FS.
func NewDashboardHandler(database *db.DB, cfg *config.Config, webFS fs.FS) *DashboardHandler {
	funcMap := template.FuncMap{
		"formatTime": func(t string) string {
			formats := []string{
				time.RFC3339,
				"2006-01-02T15:04:05Z",
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05.999999999Z07:00",
			}
			for _, f := range formats {
				if parsed, err := time.Parse(f, t); err == nil {
					return parsed.Format("Jan 2, 2006 3:04 PM")
				}
			}
			return t
		},
		"statusColor": func(s string) string {
			switch s {
			case "pending":
				return "yellow"
			case "completed":
				return "green"
			case "cancelled":
				return "gray"
			case "timeout":
				return "red"
			default:
				return "gray"
			}
		},
		"priorityColor": func(p string) string {
			switch p {
			case "critical":
				return "red"
			case "high":
				return "orange"
			case "medium":
				return "blue"
			case "low":
				return "gray"
			default:
				return "gray"
			}
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"json": func(v interface{}) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
		"deref": func(v *string) string {
			if v != nil {
				return *v
			}
			return ""
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"upper": func(s string) string { return strings.ToUpper(s) },
		"derefFloat": func(v interface{}) float64 {
			switch f := v.(type) {
			case *float64:
				if f != nil {
					return *f
				}
				return 0
			case float64:
				return f
			default:
				return 0
			}
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"renderIcon": func(icon string, size string) template.HTML {
			// If it looks like an emoji (contains non-ASCII), render as-is
			for _, r := range icon {
				if r > 127 {
					return template.HTML(template.HTMLEscapeString(icon))
				}
			}
			// Otherwise it's an icon name — map to Heroicon SVG
			svgClass := "h-6 w-6"
			if size == "lg" {
				svgClass = "h-10 w-10"
			}
			icons := map[string]string{
				"shield-check":     `<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z"/>`,
				"code-bracket":     `<path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75 22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3-4.5 16.5"/>`,
				"cube":             `<path stroke-linecap="round" stroke-linejoin="round" d="m21 7.5-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9"/>`,
				"bolt":             `<path stroke-linecap="round" stroke-linejoin="round" d="m3.75 13.5 10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75Z"/>`,
				"cog-6-tooth":      `<path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z"/><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"/>`,
				"chat-bubble-left": `<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.76c0 1.6 1.123 2.994 2.707 3.227 1.068.157 2.148.279 3.238.364.466.037.893.281 1.153.671L12 21l2.652-3.978c.26-.39.687-.634 1.153-.67 1.09-.086 2.17-.208 3.238-.365 1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z"/>`,
				"bell":             `<path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0"/>`,
				"document-text":    `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"/>`,
				"check-circle":     `<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"/>`,
				"users":            `<path stroke-linecap="round" stroke-linejoin="round" d="M15 19.128a9.38 9.38 0 0 0 2.625.372 9.337 9.337 0 0 0 4.121-.952 4.125 4.125 0 0 0-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 0 1 8.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0 1 11.964-3.07M12 6.375a3.375 3.375 0 1 1-6.75 0 3.375 3.375 0 0 1 6.75 0Zm8.25 2.25a2.625 2.625 0 1 1-5.25 0 2.625 2.625 0 0 1 5.25 0Z"/>`,
				"heart":            `<path stroke-linecap="round" stroke-linejoin="round" d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12Z"/>`,
				"star":             `<path stroke-linecap="round" stroke-linejoin="round" d="M11.48 3.499a.562.562 0 0 1 1.04 0l2.125 5.111a.563.563 0 0 0 .475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 0 0-.182.557l1.285 5.385a.562.562 0 0 1-.84.61l-4.725-2.885a.562.562 0 0 0-.586 0L6.982 20.54a.562.562 0 0 1-.84-.61l1.285-5.386a.562.562 0 0 0-.182-.557l-4.204-3.602a.562.562 0 0 1 .321-.988l5.518-.442a.563.563 0 0 0 .475-.345L11.48 3.5Z"/>`,
			}
			path, ok := icons[icon]
			if !ok {
				// Unknown icon name — render a generic circle icon
				path = `<path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"/>`
			}
			return template.HTML(fmt.Sprintf(`<svg class="%s text-indigo-600" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">%s</svg>`, svgClass, path))
		},
		"bytesToString": func(v interface{}) string {
			switch b := v.(type) {
			case []byte:
				return string(b)
			case json.RawMessage:
				return string(b)
			default:
				return fmt.Sprintf("%v", v)
			}
		},
	}

	templates := make(map[string]*template.Template)

	// Pages that use the base layout
	pageNames := []string{
		"dashboard",
		"loops",
		"loop_detail",
		"loop_new",
		"api_keys",
		"requests",
		"request_detail",
	}
	for _, name := range pageNames {
		tmpl, err := template.New("").Funcs(funcMap).ParseFS(webFS,
			"templates/layouts/base.html",
			"templates/pages/"+name+".html",
		)
		if err != nil {
			slog.Error("failed to parse template", "name", name, "error", err)
			continue
		}
		templates[name] = tmpl
	}

	// Login page is standalone (no base layout)
	loginTmpl, err := template.New("").Funcs(funcMap).ParseFS(webFS, "templates/pages/login.html")
	if err != nil {
		slog.Error("failed to parse login template", "error", err)
	} else {
		templates["login"] = loginTmpl
	}

	return &DashboardHandler{
		db:        database,
		cfg:       cfg,
		templates: templates,
	}
}

// RequireSession is a middleware that checks for a valid session cookie.
// Unlike the API middleware, this redirects to /login instead of returning JSON errors.
func (dh *DashboardHandler) RequireSession(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("handoff_session")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(dh.cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := dh.db.GetUserByID(r.Context(), userID)
		if err != nil || user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), ContextKeyUser, user)
		handler(w, r.WithContext(ctx))
	}
}

// render executes a named template with the given data.
func (dh *DashboardHandler) render(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := dh.templates[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	execName := "base"
	if name == "login" {
		execName = "login"
	}

	if err := tmpl.ExecuteTemplate(w, execName, data); err != nil {
		slog.Error("template render error", "template", name, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// RootRedirect sends users to /dashboard if logged in, otherwise /login.
func (dh *DashboardHandler) RootRedirect(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("handoff_session")
	if err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// LoginPage renders the login page.
func (dh *DashboardHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	dh.render(w, "login", nil)
}

// DashboardPage renders the main dashboard overview.
func (dh *DashboardHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	user := RequireDashboardUser(w, r)
	if user == nil {
		return
	}

	stats, err := dh.db.GetDashboardStats(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to get dashboard stats", "error", err)
	}

	recent, err := dh.db.GetRecentRequests(r.Context(), user.ID, 10)
	if err != nil {
		slog.Error("failed to get recent requests", "error", err)
	}

	dh.render(w, "dashboard", DashboardPageData{
		PageData:       PageData{ActiveNav: "dashboard", User: user},
		Stats:          stats,
		RecentRequests: recent,
	})
}

// LoopsPage renders the loops listing page.
func (dh *DashboardHandler) LoopsPage(w http.ResponseWriter, r *http.Request) {
	user := RequireDashboardUser(w, r)
	if user == nil {
		return
	}

	loops, err := dh.db.ListLoops(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to list loops", "error", err)
	}
	if loops == nil {
		loops = []models.LoopWithMeta{}
	}

	dh.render(w, "loops", LoopsPageData{
		PageData: PageData{ActiveNav: "loops", User: user},
		Loops:    loops,
	})
}

// LoopNewPage renders the new loop form (GET) or creates a loop (POST).
func (dh *DashboardHandler) LoopNewPage(w http.ResponseWriter, r *http.Request) {
	user := RequireDashboardUser(w, r)
	if user == nil {
		return
	}

	if r.Method == http.MethodGet {
		dh.render(w, "loop_new", PageData{ActiveNav: "loops", User: user})
		return
	}

	// POST: create loop
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	icon := strings.TrimSpace(r.FormValue("icon"))

	if name == "" || len(name) > 100 {
		http.Error(w, "Name is required and must be 1-100 characters", http.StatusBadRequest)
		return
	}
	if icon == "" {
		icon = "circle"
	}

	inviteCode, err := GenerateUniqueInviteCode(r.Context(), dh.db)
	if err != nil {
		slog.Error("failed to generate invite code", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	now := models.NowUTC()
	loop := &models.Loop{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Icon:        icon,
		CreatorID:   user.ID,
		InviteCode:  inviteCode,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := dh.db.CreateLoop(r.Context(), loop); err != nil {
		slog.Error("failed to create loop", "error", err)
		http.Error(w, "Failed to create loop", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/loops/"+loop.ID, http.StatusSeeOther)
}

// LoopDetailPage renders the detail page for a single loop.
func (dh *DashboardHandler) LoopDetailPage(w http.ResponseWriter, r *http.Request) {
	user := RequireDashboardUser(w, r)
	if user == nil {
		return
	}

	loopID := r.PathValue("id")
	if loopID == "" {
		http.Error(w, "Missing loop ID", http.StatusBadRequest)
		return
	}

	loop := FetchLoopOrError(w, r, dh.db, loopID)
	if loop == nil {
		return
	}

	if !EnforceDashboardMembership(w, r, dh.db, loopID, user.ID) {
		return
	}

	members, err := dh.db.GetLoopMembers(r.Context(), loopID)
	if err != nil {
		slog.Error("failed to get loop members", "error", err)
	}

	requests, _, err := dh.db.ListRequests(r.Context(), user.ID, "", "", loopID, "created_at_desc", 10, 0)
	if err != nil {
		slog.Error("failed to list loop requests", "error", err)
	}

	inviteQR := generateInviteQR(dh.cfg.BaseURL, loop.InviteCode)

	dh.render(w, "loop_detail", LoopDetailPageData{
		PageData: PageData{ActiveNav: "loops", User: user},
		Loop:     loop,
		InviteQR: inviteQR,
		Members:  members,
		Requests: requests,
	})
}

// APIKeysPage renders the API keys management page.
func (dh *DashboardHandler) APIKeysPage(w http.ResponseWriter, r *http.Request) {
	user := RequireDashboardUser(w, r)
	if user == nil {
		return
	}

	keys, err := dh.db.ListAPIKeys(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to list api keys", "error", err)
	}
	if keys == nil {
		keys = []models.APIKey{}
	}

	loops, err := dh.db.ListLoops(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to list loops for api keys", "error", err)
	}

	newKey := r.URL.Query().Get("new_key")

	dh.render(w, "api_keys", APIKeysPageData{
		PageData: PageData{ActiveNav: "api_keys", User: user},
		Keys:     keys,
		Loops:    loops,
		NewKey:   newKey,
	})
}

// CreateAPIKey handles POST to create a new API key from the dashboard.
func (dh *DashboardHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user := RequireDashboardUser(w, r)
	if user == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = "Dashboard Key"
	}

	permissions := r.Form["permissions"]
	if len(permissions) == 0 {
		permissions = []string{"loops:read", "loops:write", "requests:read", "requests:write"}
	}

	var agentName *string
	if an := strings.TrimSpace(r.FormValue("agent_name")); an != "" {
		agentName = &an
	}

	allowedLoops := r.Form["allowed_loops"]

	rawKey, keyHash, err := GenerateRawAPIKey()
	if err != nil {
		slog.Error("failed to generate api key", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	apiKey := NewAPIKeyModel(user.ID, name, keyHash, rawKey[:8], permissions, agentName, allowedLoops)

	if err := dh.db.CreateAPIKey(r.Context(), apiKey); err != nil {
		slog.Error("failed to create api key", "error", err)
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/api-keys?new_key="+rawKey, http.StatusSeeOther)
}

// RevokeAPIKey handles POST to revoke an API key from the dashboard.
func (dh *DashboardHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	user := RequireDashboardUser(w, r)
	if user == nil {
		return
	}

	keyID := r.PathValue("id")
	if keyID == "" {
		http.Error(w, "Missing key ID", http.StatusBadRequest)
		return
	}

	if err := dh.db.RevokeAPIKey(r.Context(), keyID, user.ID); err != nil {
		slog.Error("failed to revoke api key", "error", err)
	}

	http.Redirect(w, r, "/api-keys", http.StatusSeeOther)
}

// RequestsPage renders the requests listing page with filters and pagination.
func (dh *DashboardHandler) RequestsPage(w http.ResponseWriter, r *http.Request) {
	user := RequireDashboardUser(w, r)
	if user == nil {
		return
	}

	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	loopID := r.URL.Query().Get("loop_id")
	pageStr := r.URL.Query().Get("page")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	const perPage = 20
	offset := (page - 1) * perPage

	requests, total, err := dh.db.ListRequests(r.Context(), user.ID, status, priority, loopID, "created_at_desc", perPage, offset)
	if err != nil {
		slog.Error("failed to list requests", "error", err)
	}

	loops, err := dh.db.ListLoops(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to list loops for filter", "error", err)
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	dh.render(w, "requests", RequestsPageData{
		PageData:   PageData{ActiveNav: "requests", User: user},
		Requests:   requests,
		Loops:      loops,
		Status:     status,
		Priority:   priority,
		LoopID:     loopID,
		Page:       page,
		TotalPages: totalPages,
		Total:      total,
	})
}

// RequestDetailPage renders the detail page for a single request.
func (dh *DashboardHandler) RequestDetailPage(w http.ResponseWriter, r *http.Request) {
	user := RequireDashboardUser(w, r)
	if user == nil {
		return
	}

	reqID := r.PathValue("id")
	if reqID == "" {
		http.Error(w, "Missing request ID", http.StatusBadRequest)
		return
	}

	request := FetchRequestOrError(w, r, dh.db, reqID)
	if request == nil {
		return
	}

	if !EnforceDashboardMembership(w, r, dh.db, request.LoopID, user.ID) {
		return
	}

	dh.render(w, "request_detail", RequestDetailPageData{
		PageData: PageData{ActiveNav: "requests", User: user},
		Request:  request,
	})
}

// generateInviteQR generates a base64-encoded QR code PNG for a loop invite link.
func generateInviteQR(baseURL, inviteCode string) string {
	link := fmt.Sprintf("%s/join/%s", baseURL, inviteCode)
	qrPNG, err := qrcode.Encode(link, qrcode.Medium, 256)
	if err != nil {
		slog.Error("failed to generate QR code", "error", err)
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG)
}

