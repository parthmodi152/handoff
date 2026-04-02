# Handoff

Mobile-first human oversight for AI workflows. Your AI agents send approval requests — you review, approve, and respond from your phone with real-time push notifications and Live Activities.

## Screenshots

| Home | Live Activity | Lock Screen Widget |
|------|--------------|-------------------|
| ![Home](screenshots/home.png) | ![Live Activity](screenshots/live-activity-expanded.png) | ![Widget](screenshots/live-activity-compact.png) |

## Structure

```
ios/        # Swift iOS app with Live Activities & widgets
server/     # Go backend with APNs push, SSE, webhooks
screenshots/
design/     # Pencil design files
```

## Server Setup

```bash
cd server
cp .env.example .env  # configure your env
go run ./cmd/handoff/
# Server starts on :8080
```

## iOS

Open `ios/Handoff.xcodeproj` in Xcode and run.

---

## Using Handoff in Agentic Workflows

Handoff lets AI agents pause and ask a human for input — approvals, choices, ratings, text responses — then resume once the human responds. Auth is via API keys created in the dashboard.

### Authentication

All API calls use Bearer token auth with an API key:

```
Authorization: Bearer hnd_your_api_key_here
```

Create API keys in the web dashboard at `/api-keys`. Keys can be scoped to specific loops and given an `agent_name` for traceability.

### Quick Start: Ask a Human for Approval

```bash
# 1. Create a boolean (yes/no) approval request
curl -X POST http://localhost:8080/api/v1/loops/LOOP_ID/requests \
  -H "Authorization: Bearer hnd_your_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "processing_type": "time-sensitive",
    "type": "markdown",
    "title": "Deploy v2.1 to production?",
    "request_text": "All tests pass. 3 new features, 2 bug fixes. Changelog attached.",
    "response_type": "boolean",
    "response_config": {"true_label": "Approve", "false_label": "Reject"},
    "priority": "high",
    "timeout_seconds": 300
  }'

# Response:
# { "data": { "request_id": "abc-123", "status": "pending", "timeout_at": "..." } }

# 2. Wait for the human to respond (SSE stream)
curl -N http://localhost:8080/api/v1/requests/abc-123/events \
  -H "Authorization: Bearer hnd_your_api_key"

# SSE event fires when human responds:
# event: completed
# data: {"request_id":"abc-123","status":"completed","request":{...,"response_data":{"boolean":true,"boolean_label":"Approve"}}}

# 3. Or poll instead of streaming
curl http://localhost:8080/api/v1/requests/abc-123 \
  -H "Authorization: Bearer hnd_your_api_key"
```

The human gets a push notification on their phone, reviews it, taps Approve/Reject, and your agent gets the answer instantly via SSE.

### Response Types

Handoff supports 6 response types. Each requires a `response_config` that defines the UI:

#### Boolean (Yes/No)
```json
{
  "response_type": "boolean",
  "response_config": {
    "true_label": "Approve",
    "false_label": "Reject"
  }
}
```
Response: `{"boolean": true, "boolean_label": "Approve"}`

#### Single Select
```json
{
  "response_type": "single_select",
  "response_config": {
    "options": [
      {"value": "fix", "label": "Fix the bug"},
      {"value": "skip", "label": "Skip for now"},
      {"value": "escalate", "label": "Escalate to team lead"}
    ]
  }
}
```
Response: `{"selected_value": "fix", "selected_label": "Fix the bug"}`

#### Multi Select
```json
{
  "response_type": "multi_select",
  "response_config": {
    "options": [
      {"value": "typo", "label": "Fix typos"},
      {"value": "format", "label": "Reformat code"},
      {"value": "tests", "label": "Add tests"}
    ],
    "min_selections": 1,
    "max_selections": 3
  }
}
```
Response: `{"selected_values": ["typo", "tests"], "selected_labels": ["Fix typos", "Add tests"]}`

#### Text
```json
{
  "response_type": "text",
  "response_config": {
    "placeholder": "Describe the issue...",
    "max_length": 500
  }
}
```
Response: `"The header image is too large, compress it to under 200KB"`

#### Number
```json
{
  "response_type": "number",
  "response_config": {
    "min_value": 0,
    "max_value": 10000,
    "decimal_places": 2
  }
}
```
Response: `{"number": 49.99, "formatted_value": "49.99"}`

#### Rating
```json
{
  "response_type": "rating",
  "response_config": {
    "scale_min": 1,
    "scale_max": 5,
    "labels": {"1": "Terrible", "5": "Excellent"}
  }
}
```
Response: `{"rating": 4, "rating_label": "4"}`

### Python Example: Agent with Human Approval

```python
import requests
import sseclient
import json

HANDOFF_URL = "http://localhost:8080"
API_KEY = "hnd_your_api_key"
LOOP_ID = "your-loop-id"
HEADERS = {"Authorization": f"Bearer {API_KEY}", "Content-Type": "application/json"}

def ask_human(title, description, response_type="boolean", response_config=None, timeout=300, priority="medium"):
    """Send a request to a human and wait for their response."""
    if response_config is None:
        response_config = {"true_label": "Approve", "false_label": "Reject"}

    # Create the request
    resp = requests.post(f"{HANDOFF_URL}/api/v1/loops/{LOOP_ID}/requests", headers=HEADERS, json={
        "processing_type": "time-sensitive",
        "type": "markdown",
        "title": title,
        "request_text": description,
        "response_type": response_type,
        "response_config": response_config,
        "priority": priority,
        "timeout_seconds": timeout,
    })
    request_id = resp.json()["data"]["request_id"]

    # Wait for response via SSE
    stream = requests.get(f"{HANDOFF_URL}/api/v1/requests/{request_id}/events",
                          headers=HEADERS, stream=True)
    client = sseclient.SSEClient(stream)
    for event in client.events():
        data = json.loads(event.data)
        return data["request"]["status"], data["request"].get("response_data")

# --- Use in your agent ---

status, response = ask_human(
    title="Deploy v2.1 to production?",
    description="All 47 tests pass. Changes: new auth flow, bug fix for #482.",
    priority="high",
    timeout=300,
)

if status == "completed" and response.get("boolean"):
    print("Human approved — deploying!")
    # deploy()
elif status == "timeout":
    print("No response in time — using default")
else:
    print("Human rejected — aborting")
```

### Webhook Callbacks

Instead of polling or SSE, pass a `callback_url` to get notified when the human responds:

```json
{
  "title": "Review flagged content",
  "request_text": "User posted potentially inappropriate content...",
  "response_type": "single_select",
  "response_config": {
    "options": [
      {"value": "approve", "label": "Content is fine"},
      {"value": "remove", "label": "Remove content"},
      {"value": "ban", "label": "Remove & ban user"}
    ]
  },
  "callback_url": "https://your-agent.com/webhook/handoff",
  "processing_type": "time-sensitive",
  "type": "markdown",
  "timeout_seconds": 600
}
```

The webhook POSTs the full response to your URL when the human acts.

### API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/loops/{loop_id}/requests` | Create a review request |
| `GET` | `/api/v1/requests/{id}` | Get request status & response |
| `GET` | `/api/v1/requests/{id}/events` | SSE stream for real-time updates |
| `DELETE` | `/api/v1/requests/{id}` | Cancel a pending request |
| `GET` | `/api/v1/requests?status=pending&loop_id=...` | List requests with filters |
| `GET` | `/api/v1/loops` | List your loops |
| `GET` | `/api/v1/test` | Test your API key |

### Request Fields

| Field | Required | Description |
|-------|----------|-------------|
| `processing_type` | yes | `"time-sensitive"` (has timeout) or `"deferred"` |
| `type` | yes | `"markdown"` or `"image"` |
| `title` | yes | Short title (1-200 chars) |
| `request_text` | yes | Full description (1-10000 chars, supports markdown) |
| `response_type` | yes | `boolean`, `single_select`, `multi_select`, `text`, `number`, `rating` |
| `response_config` | yes | Config for the response UI (see examples above) |
| `priority` | no | `low`, `medium` (default), `high`, `critical` |
| `timeout_seconds` | time-sensitive | 60–604800 seconds (1 min – 7 days) |
| `callback_url` | no | Webhook URL for async notification |
| `default_response` | no | Auto-response if timeout expires |
| `context` | no | Arbitrary JSON metadata (up to 50KB) |
| `image_url` | if type=image | URL of the image to review |
