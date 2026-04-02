package push

import (
	"strings"

	"github.com/sideshow/apns2/payload"
)

// TruncateText limits text to maxLen runes with ellipsis.
func TruncateText(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// displayText returns title if non-empty, otherwise truncated requestText.
func displayText(title, requestText string, maxLen int) string {
	if title != "" {
		return TruncateText(title, maxLen)
	}
	return TruncateText(requestText, maxLen)
}

// NewRequestPayload builds a push notification for a new approval request.
// trueLabel/falseLabel are only used for boolean response types (pass "" for others).
func NewRequestPayload(title, requestText, loopName, responseType, requestID, loopID, trueLabel, falseLabel string) *payload.Payload {
	p := payload.NewPayload().
		AlertTitle("New Request in " + loopName).
		AlertBody(displayText(title, requestText, 200)).
		Sound("default").
		Category("NEW_REQUEST_" + strings.ToUpper(responseType)).
		ThreadID("loop-" + loopID).
		ContentAvailable().
		Custom("request_id", requestID).
		Custom("loop_id", loopID).
		Custom("response_type", responseType)

	if trueLabel != "" {
		p = p.Custom("true_label", trueLabel)
	}
	if falseLabel != "" {
		p = p.Custom("false_label", falseLabel)
	}
	return p
}

// CompletedPayload builds a push notification for a completed request.
func CompletedPayload(title, requestText, loopName, responderName, requestID, loopID string) *payload.Payload {
	return payload.NewPayload().
		AlertTitle("Request Completed").
		AlertBody(responderName + " responded to '" + displayText(title, requestText, 100) + "'").
		AlertSubtitle(loopName).
		Sound("default").
		Category("REQUEST_COMPLETED").
		ThreadID("loop-" + loopID).
		ContentAvailable().
		Custom("request_id", requestID).
		Custom("loop_id", loopID)
}

// CancelledPayload builds a push notification for a cancelled request.
func CancelledPayload(title, requestText, loopName, requestID, loopID string) *payload.Payload {
	return payload.NewPayload().
		AlertTitle("Request Cancelled").
		AlertBody("'" + displayText(title, requestText, 150) + "' was cancelled").
		AlertSubtitle(loopName).
		Sound("default").
		Category("REQUEST_CANCELLED").
		ThreadID("loop-" + loopID).
		ContentAvailable().
		Custom("request_id", requestID).
		Custom("loop_id", loopID)
}

// TimedOutPayload builds a push notification for a timed-out request.
func TimedOutPayload(title, requestText, loopName, requestID, loopID string) *payload.Payload {
	return payload.NewPayload().
		AlertTitle("Request Timed Out").
		AlertBody("'" + displayText(title, requestText, 150) + "' expired without response").
		AlertSubtitle(loopName).
		Sound("default").
		Category("REQUEST_TIMED_OUT").
		ThreadID("loop-" + loopID).
		ContentAvailable().
		Custom("request_id", requestID).
		Custom("loop_id", loopID)
}
