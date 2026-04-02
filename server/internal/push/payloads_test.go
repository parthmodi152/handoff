package push

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateText_Short(t *testing.T) {
	got := TruncateText("hello", 10)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestTruncateText_ExactLength(t *testing.T) {
	got := TruncateText("12345", 5)
	if got != "12345" {
		t.Errorf("got %q, want %q", got, "12345")
	}
}

func TestTruncateText_Long(t *testing.T) {
	got := TruncateText("hello world this is a long text", 10)
	// 9 runes + "…" = 10 runes
	runes := []rune(got)
	if len(runes) > 10 {
		t.Errorf("truncated text too long: %q (runes=%d)", got, len(runes))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis suffix, got %q", got)
	}
	if got != "hello wor…" {
		t.Errorf("got %q, want %q", got, "hello wor…")
	}
}

// TestTruncateText_Unicode verifies rune-safe truncation for multi-byte characters.
func TestTruncateText_Unicode(t *testing.T) {
	// Emoji: "🚀🎉" = 2 runes. maxLen=2 should return as-is.
	got := TruncateText("🚀🎉", 2)
	if got != "🚀🎉" {
		t.Errorf("2-rune emoji with maxLen=2: got %q, want %q", got, "🚀🎉")
	}

	// maxLen=1 → take 0 runes + ellipsis = "…"
	got = TruncateText("🚀🎉", 1)
	if !utf8.ValidString(got) {
		t.Errorf("invalid UTF-8: %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis, got %q", got)
	}

	// CJK: "你好世界" = 4 runes. maxLen=3 → "你好…"
	got = TruncateText("你好世界", 3)
	if !utf8.ValidString(got) {
		t.Errorf("invalid UTF-8 for CJK: %q", got)
	}
	if got != "你好…" {
		t.Errorf("CJK truncation: got %q, want %q", got, "你好…")
	}

	// Mixed: "ab🚀cd" = 5 runes. maxLen=4 → "ab🚀…"
	got = TruncateText("ab🚀cd", 4)
	if !utf8.ValidString(got) {
		t.Errorf("invalid UTF-8 for mixed: %q", got)
	}
	if got != "ab🚀…" {
		t.Errorf("mixed truncation: got %q, want %q", got, "ab🚀…")
	}
}

func TestNewRequestPayload_Boolean(t *testing.T) {
	p := NewRequestPayload("Deploy Title", "Deploy v2.3?", "Prod Loop", "boolean", "req-1", "loop-1", "Approve", "Reject")
	raw, _ := p.MarshalJSON()

	var m map[string]interface{}
	json.Unmarshal(raw, &m)

	aps := m["aps"].(map[string]interface{})
	alert := aps["alert"].(map[string]interface{})

	if title, _ := alert["title"].(string); title != "New Request in Prod Loop" {
		t.Errorf("title = %q", title)
	}
	if body, _ := alert["body"].(string); body != "Deploy Title" {
		t.Errorf("body = %q, want %q", body, "Deploy Title")
	}
	if cat, _ := aps["category"].(string); cat != "NEW_REQUEST_BOOLEAN" {
		t.Errorf("category = %q, want NEW_REQUEST_BOOLEAN", cat)
	}
	if ca, ok := aps["content-available"].(float64); !ok || ca != 1 {
		t.Errorf("content-available = %v, want 1", aps["content-available"])
	}
	if rid, _ := m["request_id"].(string); rid != "req-1" {
		t.Errorf("request_id = %q", rid)
	}
	if lid, _ := m["loop_id"].(string); lid != "loop-1" {
		t.Errorf("loop_id = %q", lid)
	}
	if tl, _ := m["true_label"].(string); tl != "Approve" {
		t.Errorf("true_label = %q, want Approve", tl)
	}
	if fl, _ := m["false_label"].(string); fl != "Reject" {
		t.Errorf("false_label = %q, want Reject", fl)
	}
}

func TestNewRequestPayload_Text(t *testing.T) {
	p := NewRequestPayload("ETA Check", "What's the ETA?", "Dev Loop", "text", "req-2", "loop-2", "", "")
	raw, _ := p.MarshalJSON()

	var m map[string]interface{}
	json.Unmarshal(raw, &m)

	aps := m["aps"].(map[string]interface{})
	if cat, _ := aps["category"].(string); cat != "NEW_REQUEST_TEXT" {
		t.Errorf("category = %q, want NEW_REQUEST_TEXT", cat)
	}
	// No boolean labels for text type
	if _, exists := m["true_label"]; exists {
		t.Error("true_label should not be present for text type")
	}
	if _, exists := m["false_label"]; exists {
		t.Error("false_label should not be present for text type")
	}
}

func TestCompletedPayload(t *testing.T) {
	p := CompletedPayload("Deploy Title", "Deploy v2.3?", "Prod Loop", "Alice", "req-1", "loop-1")
	raw, _ := p.MarshalJSON()

	var m map[string]interface{}
	json.Unmarshal(raw, &m)

	aps := m["aps"].(map[string]interface{})
	alert := aps["alert"].(map[string]interface{})

	if title, _ := alert["title"].(string); title != "Request Completed" {
		t.Errorf("title = %q", title)
	}
	body, _ := alert["body"].(string)
	if !strings.Contains(body, "Alice") {
		t.Errorf("body should contain responder name, got %q", body)
	}
	if cat, _ := aps["category"].(string); cat != "REQUEST_COMPLETED" {
		t.Errorf("category = %q", cat)
	}
	if ca, ok := aps["content-available"].(float64); !ok || ca != 1 {
		t.Errorf("content-available = %v, want 1", aps["content-available"])
	}
}

func TestCancelledPayload(t *testing.T) {
	p := CancelledPayload("Deploy Title", "Deploy v2.3?", "Prod Loop", "req-1", "loop-1")
	raw, _ := p.MarshalJSON()

	var m map[string]interface{}
	json.Unmarshal(raw, &m)

	aps := m["aps"].(map[string]interface{})
	if cat, _ := aps["category"].(string); cat != "REQUEST_CANCELLED" {
		t.Errorf("category = %q", cat)
	}
	if ca, ok := aps["content-available"].(float64); !ok || ca != 1 {
		t.Errorf("content-available = %v, want 1", aps["content-available"])
	}
}

func TestTimedOutPayload(t *testing.T) {
	p := TimedOutPayload("Deploy Title", "Deploy v2.3?", "Prod Loop", "req-1", "loop-1")
	raw, _ := p.MarshalJSON()

	var m map[string]interface{}
	json.Unmarshal(raw, &m)

	aps := m["aps"].(map[string]interface{})
	if cat, _ := aps["category"].(string); cat != "REQUEST_TIMED_OUT" {
		t.Errorf("category = %q", cat)
	}
	alert := aps["alert"].(map[string]interface{})
	body, _ := alert["body"].(string)
	if !strings.Contains(body, "expired") {
		t.Errorf("body should mention 'expired', got %q", body)
	}
	if ca, ok := aps["content-available"].(float64); !ok || ca != 1 {
		t.Errorf("content-available = %v, want 1", aps["content-available"])
	}
}
