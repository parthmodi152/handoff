package push

import (
	"encoding/json"
	"time"

	"github.com/hitl-sh/handoff-server/internal/models"
)

const AppleReferenceEpochOffset int64 = 978307200

// BuildLiveActivityPayload builds the raw JSON for a Live Activity "update" push.
func BuildLiveActivityPayload(contentState interface{}, staleDateUnix *int64, alertTitle, alertBody string) ([]byte, error) {
	aps := map[string]interface{}{
		"timestamp":     time.Now().Unix(),
		"event":         "update",
		"content-state": contentState,
	}
	if staleDateUnix != nil {
		aps["stale-date"] = *staleDateUnix
	}
	if alertTitle != "" {
		aps["alert"] = map[string]string{
			"title": alertTitle,
			"body":  alertBody,
		}
		aps["sound"] = "default"
	}
	return json.Marshal(map[string]interface{}{"aps": aps})
}

// BuildLiveActivityEndPayload builds the raw JSON for a Live Activity "end" push.
func BuildLiveActivityEndPayload(contentState interface{}, dismissalDateUnix int64) ([]byte, error) {
	aps := map[string]interface{}{
		"timestamp":      time.Now().Unix(),
		"event":          "end",
		"content-state":  contentState,
		"dismissal-date": dismissalDateUnix,
	}
	return json.Marshal(map[string]interface{}{"aps": aps})
}

// BuildLoopContentState builds the ContentState matching iOS HandoffLoopAttributes.ContentState.
// Requests must already be sorted (by priority then timeout) from the SQL query.
func BuildLoopContentState(requests []models.Request, totalPending int) map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(requests))
	for _, req := range requests {
		item := map[string]interface{}{
			"id":           req.ID,
			"text":         displayText(req.Title, req.RequestText, 60),
			"responseType": req.ResponseType,
			"priority":     req.Priority,
		}
		if req.TimeoutAt != nil {
			t, err := time.Parse(time.RFC3339, *req.TimeoutAt)
			if err == nil {
				item["timeoutAt"] = float64(t.Unix()) - float64(AppleReferenceEpochOffset)
			} else {
				item["timeoutAt"] = nil
			}
		} else {
			item["timeoutAt"] = nil
		}
		items = append(items, item)
	}
	return map[string]interface{}{
		"pendingRequests": items,
		"totalPending":    totalPending,
	}
}

// EarliestTimeoutUnix returns the earliest timeout_at as Unix timestamp for stale-date.
func EarliestTimeoutUnix(requests []models.Request) *int64 {
	var earliest *time.Time
	for _, req := range requests {
		if req.TimeoutAt != nil {
			t, err := time.Parse(time.RFC3339, *req.TimeoutAt)
			if err == nil && (earliest == nil || t.Before(*earliest)) {
				earliest = &t
			}
		}
	}
	if earliest == nil {
		return nil
	}
	unix := earliest.Unix()
	return &unix
}
