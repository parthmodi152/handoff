# Handoff Server

A self-hosted Human-in-the-Loop server for adding human oversight to AI workflows. Single Go binary backed by SQLite — zero external dependencies. Includes a web dashboard for managing loops, API keys, and reviewing requests.

## Quick Start

### Prerequisites

- Go 1.23+

### Build & Run

```bash
go build -o handoff ./cmd/handoff/

# Start in dev mode (enables dev login, no OAuth needed)
HANDOFF_DEV_MODE=true HANDOFF_JWT_SECRET=dev-secret ./handoff
```

The server starts on `http://localhost:8080`. The SQLite database is created automatically.

### Dashboard

Open `http://localhost:8080/login` in your browser. In dev mode, use `POST /auth/dev-login` to get a session token, then set the `handoff_session` cookie. In production, sign in with Google or Apple OAuth.

The dashboard provides:
- **Dashboard** (`/dashboard`) — Stats overview, recent requests, quick actions
- **Loops** (`/loops`) — Create and manage loops, view invite codes and QR codes
- **Requests** (`/requests`) — Browse requests with status/priority/loop filters
- **API Keys** (`/api-keys`) — Create, copy, and revoke API keys

### Agent Approval Workflow

This is the typical flow for adding human approval gates to AI agent workflows:

**Step 1: Set up (one-time, via dashboard or API)**

```bash
# Get a JWT (dev mode)
TOKEN=$(curl -s -X POST http://localhost:8080/auth/dev-login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "name": "Admin"}' | jq -r '.data.token')

# Create a loop (a channel for requests)
LOOP_ID=$(curl -s -X POST http://localhost:8080/api/v1/loops \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Deploy Approvals", "description": "Production deploy gates", "icon": "🚀"}' \
  | jq -r '.data.loop.id')

# Create an API key for your agent
API_KEY=$(curl -s -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "CI Agent", "permissions": ["requests:read", "requests:write", "loops:read"]}' \
  | jq -r '.data.api_key.key')
```

**Step 2: Agent sends an approval request**

```bash
REQUEST_ID=$(curl -s -X POST "http://localhost:8080/api/v1/loops/$LOOP_ID/requests" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "request_text": "Deploy v2.3.1 to production?",
    "type": "markdown",
    "response_type": "boolean",
    "processing_type": "time-sensitive",
    "timeout_seconds": 3600,
    "priority": "high",
    "response_config": {
      "true_label": "Approve",
      "false_label": "Reject"
    }
  }' | jq -r '.data.request_id')
```

**Step 3: Agent waits for the response (SSE)**

Open a Server-Sent Events connection to receive the result in real-time:

```bash
curl -N "http://localhost:8080/api/v1/requests/$REQUEST_ID/events" \
  -H "Authorization: Bearer $API_KEY"
```

The connection stays open until the request is resolved. When a human responds, cancels, or the request times out, you receive a single SSE event and the connection closes:

```
event: completed
data: {"request":{"id":"...","status":"completed","response_data":{"boolean":true,"boolean_label":"Approve"},"response_by":{"user_id":"...","name":"Admin"},"response_at":"2026-02-28T...","response_time_seconds":14.0}}
```

If the request is already resolved when you connect, the event is sent immediately.

**Step 4: Human responds** via the dashboard (`/requests/<id>`) or mobile app.

### Agent Integration Patterns

There are three ways an agent can interact with Handoff, depending on the use case:

#### Pattern 1: Synchronous Wait (SSE)

Best for time-sensitive approvals where the agent should block until a human responds.

```bash
# Create request, then immediately open SSE connection
REQUEST_ID=$(curl -s -X POST ".../requests" ... | jq -r '.data.request_id')
curl -N ".../requests/$REQUEST_ID/events" -H "Authorization: Bearer $API_KEY"
# Agent blocks here until human responds, request times out, or is cancelled
```

The SSE endpoint (`GET /api/v1/requests/{id}/events`) returns one event with the terminal status:
- `completed` — human responded, `response_data` included
- `cancelled` — request was cancelled by creator or loop owner
- `timeout` — request exceeded its `timeout_seconds`

#### Pattern 2: Fire-and-Forget (Webhook)

Best for deferred work where the agent doesn't need to block. Provide a `callback_url` when creating the request:

```bash
curl -X POST ".../requests" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "request_text": "Review this content",
    "response_type": "single_select",
    "processing_type": "deferred",
    "callback_url": "https://my-agent.example.com/webhook",
    "response_config": {
      "options": [
        {"value": "approve", "label": "Approve"},
        {"value": "reject", "label": "Reject"}
      ]
    }
  }'
```

When the request is resolved, Handoff POSTs to your `callback_url`:

```json
{
  "event": "request.completed",
  "request_id": "uuid",
  "timestamp": "2026-02-28T...",
  "data": {
    "request": {
      "id": "uuid",
      "status": "completed",
      "response_data": {"selected_value": "approve", "selected_label": "Approve"},
      "response_by": {"user_id": "...", "name": "Admin"}
    }
  }
}
```

Webhooks retry up to 3 times with exponential backoff (1s, 2s, 4s) on 5xx or network errors.

#### Pattern 3: Poll

If SSE isn't practical for your setup, you can poll:

```bash
curl -s "http://localhost:8080/api/v1/requests/$REQUEST_ID" \
  -H "Authorization: Bearer $API_KEY" | jq '.data.request | {status, response_data}'
```

Or list all pending requests:

```bash
curl -s "http://localhost:8080/api/v1/requests?status=pending" \
  -H "Authorization: Bearer $API_KEY" | jq '.data.requests'
```

### Quick Test (curl only)

```bash
# Health check
curl http://localhost:8080/health

# Create a dev user and get a JWT
curl -s -X POST http://localhost:8080/auth/dev-login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@dev.com", "name": "Test User"}' | jq
```

## Configuration

Copy `.env.example` to `.env` and adjust values:

| Variable | Default | Description |
|----------|---------|-------------|
| `HANDOFF_PORT` | `8080` | Server port |
| `HANDOFF_DB_PATH` | `handoff.db` | SQLite database file path |
| `HANDOFF_BASE_URL` | `http://localhost:8080` | Public URL for OAuth redirects |
| `HANDOFF_JWT_SECRET` | (insecure default) | JWT signing secret (change in production) |
| `HANDOFF_DEV_MODE` | `false` | Set `true` to enable `POST /auth/dev-login` |
| `HANDOFF_APPLE_CLIENT_ID` | | Apple Services ID |
| `HANDOFF_APPLE_CLIENT_SECRET` | | Apple client secret JWT |
| `HANDOFF_APPLE_TEAM_ID` | | Apple Developer Team ID |
| `HANDOFF_APPLE_KEY_ID` | | Apple key ID |
| `HANDOFF_GOOGLE_CLIENT_ID` | | Google OAuth client ID |
| `HANDOFF_GOOGLE_CLIENT_SECRET` | | Google OAuth client secret |
| `HANDOFF_APNS_KEY_PATH` | | Path to APNs `.p8` key file |
| `HANDOFF_APNS_KEY_ID` | | APNs key ID |
| `HANDOFF_APNS_PRODUCTION` | `false` | Use APNs production gateway |

For local development, only `HANDOFF_DEV_MODE=true` and `HANDOFF_JWT_SECRET` are needed. Push notifications and OAuth are optional — the server degrades gracefully without them.

## API

All responses use a consistent envelope:

```json
{"error": false, "msg": "Success", "data": {...}}
```

### Authentication

Two auth methods, disambiguated by token prefix:
- **JWT** (`eyJ...`): Returned after OAuth or dev login. Used as `Authorization: Bearer <jwt>` or session cookie.
- **API Key** (`hnd_...`): Created via `POST /api/v1/api-keys`. Used as `Authorization: Bearer hnd_...`.

### Dashboard Pages

| Path | Description |
|------|-------------|
| `/login` | Sign in (Google/Apple OAuth, or dev login) |
| `/dashboard` | Stats overview, recent requests |
| `/loops` | List and create loops |
| `/loops/new` | Create a new loop |
| `/loops/{id}` | Loop detail: members, invite code, QR, requests |
| `/api-keys` | Create, list, and revoke API keys |
| `/requests` | Browse requests with filters |
| `/requests/{id}` | Request detail with response data |

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | None | Health check |
| `GET` | `/auth/google` | None | Start Google OAuth |
| `GET` | `/auth/google/callback` | None | Google OAuth callback |
| `GET` | `/auth/apple` | None | Start Apple OAuth |
| `GET/POST` | `/auth/apple/callback` | None | Apple OAuth callback |
| `POST` | `/auth/dev-login` | None | Dev-only login (requires `DEV_MODE=true`) |
| `POST` | `/auth/logout` | Session | Clear session |
| `GET` | `/auth/me` | Any | Get current user |
| `GET` | `/api/v1/test` | Any | Validate auth credentials |
| `POST` | `/api/v1/api-keys` | Session | Create API key |
| `GET` | `/api/v1/api-keys` | Any | List API keys |
| `DELETE` | `/api/v1/api-keys/{id}` | Session | Revoke API key |
| `POST` | `/api/v1/loops` | Any | Create loop |
| `GET` | `/api/v1/loops` | Any | List loops |
| `GET` | `/api/v1/loops/{id}` | Any | Get loop + members |
| `POST` | `/api/v1/loops/join` | Session | Join loop by invite code |
| `POST` | `/api/v1/loops/{loop_id}/requests` | Any | Create review request |
| `GET` | `/api/v1/requests` | Any | List requests (filterable) |
| `GET` | `/api/v1/requests/{id}` | Any | Get request details |
| `DELETE` | `/api/v1/requests/{id}` | Any | Cancel pending request |
| `POST` | `/api/v1/requests/{id}/respond` | Session | Submit response |
| `GET` | `/api/v1/requests/{id}/events` | Any | SSE stream for request status changes |
| `POST` | `/api/v1/devices` | Session | Register push notification device token |
| `DELETE` | `/api/v1/devices/{token}` | Session | Unregister device token |

**Auth types**: `Session` = JWT cookie or `Authorization: Bearer <jwt>`. `Any` = JWT or API key (`X-API-Key` header or `Authorization: Bearer hnd_...`).

### Response Types

Requests support 6 response types, each with config validation and response data validation:

| Type | Config | Response Data |
|------|--------|---------------|
| `boolean` | `true_label`, `false_label` | `{"boolean": true, "boolean_label": "Yes"}` |
| `single_select` | `options: [{value, label}]` (2-20) | `{"selected_value": "a", "selected_label": "A"}` |
| `multi_select` | `options`, `min_selections`, `max_selections` | `{"selected_values": [...], "selected_labels": [...]}` |
| `text` | `min_length`, `max_length` | `"response text string"` |
| `rating` | `scale_min`, `scale_max`, `scale_step` | `{"rating": 4.0}` |
| `number` | `min_value`, `max_value`, `decimal_places` | `{"number": 42.5}` |

## Testing

```bash
# Run all tests
go test ./... -v -count=1

# Run API tests only
go test ./internal/api/ -v -count=1

# Run event broker tests
go test ./internal/events/ -v -count=1

# Run a specific test
go test ./internal/api/ -v -run TestLoops -count=1
```

Tests use an in-memory SQLite database and `httptest.Server` — no external services needed. Test files are modular:

| File | Coverage |
|------|----------|
| `testutil_test.go` | Shared helpers (test env, dev login, create helpers) |
| `auth_test.go` | Health, dev login, auth/me |
| `apikeys_test.go` | Create, list, revoke, auth validation |
| `loops_test.go` | Create, list, get, join, access control |
| `requests_test.go` | Create (all 6 types), list, get, cancel, timeout |
| `respond_test.go` | Respond (all 6 types), status lifecycle |
| `validation_test.go` | Request creation validation, response data validation |
| `devices_test.go` | Device token registration, upsert, unregister |
| `sse_test.go` | SSE streaming (already-completed, pending, auth, 404) |
| `events/broker_test.go` | Pub/sub subscribe, publish, unsubscribe, isolation |

## Project Structure

```
handoff-server/
  cmd/handoff/
    main.go               # Entry point, graceful shutdown
    migrations/
      001_initial.sql      # Schema (embedded at build time)
  internal/
    api/
      router.go            # Route registration (API + dashboard)
      dashboard.go         # Dashboard page handlers + template rendering
      auth.go              # OAuth + dev login handlers
      apikeys.go           # API key CRUD
      loops.go             # Loop CRUD + join
      requests.go          # Request CRUD
      respond.go           # Response submission
      sse.go               # SSE endpoint for real-time request events
      devices.go           # Device token registration (push notifications)
      validation.go        # All validation logic
      middleware.go         # Recovery, CORS, logging, auth
      helpers.go            # JSON response helpers
    config/
      config.go            # Environment variable loading
    db/
      db.go                # SQLite connection, migrations, queries
      devices.go           # Device token queries
      stats.go             # Dashboard stats queries
    events/
      broker.go            # In-process pub/sub for request status changes
      webhook.go           # Webhook delivery to callback URLs
    models/
      models.go            # Domain structs
    push/
      apns.go              # APNs client (token-based auth)
      sender.go            # Fan-out push notification sender
      payloads.go          # Push notification payload builders
      push.go              # Sender interface
    worker/
      timeout.go           # Background worker for expired requests
  web/
    embed.go               # Embeds templates + static files into binary
    templates/
      layouts/base.html    # Shared layout (sidebar, nav, flash messages)
      pages/               # Page templates (dashboard, loops, requests, etc.)
    static/
      css/app.css          # Custom styles
      js/app.js            # Copy-to-clipboard, flash auto-dismiss
```

## Architecture

- **Single binary**: All templates and static assets are embedded via `embed.FS` — no external files needed
- **Database**: SQLite with WAL mode, `busy_timeout=5000`, `foreign_keys=ON`
- **No CGo**: Uses `modernc.org/sqlite` (pure Go SQLite driver)
- **Auth**: Google/Apple OAuth, JWT sessions, API key auth with bcrypt hashing
- **Dashboard**: Server-rendered HTML with htmx (partial page updates), Alpine.js (interactivity), Tailwind CSS — all via CDN, no build step
- **Routing**: Go 1.22+ `net/http` with method-based patterns (`"POST /api/v1/loops"`)
- **Logging**: `log/slog` structured logging
- **Middleware**: Recovery -> RequestID -> CORS -> Logging -> Auth

## Current Status

Working:
- 23 API endpoints with full request/response lifecycle
- Web dashboard (loops, requests, API keys, stats)
- All 6 response types with config and data validation
- API key auth with bcrypt + JWT session auth
- SSE streaming for real-time request status events
- Webhook callbacks to agent `callback_url` endpoints
- APNs push notifications (token-based auth, fan-out to loop members)
- Device token registration for iOS push
- Request timeout background worker (30s polling interval)
- Dev login for local testing
- QR code generation for loop invites
- 88 passing tests

Not yet implemented:
- Apple OAuth web flow (native iOS auth works, web needs Services ID)
- Live Activities (depends on iOS implementation)
- WebSocket for bidirectional real-time (deferred until needed)
