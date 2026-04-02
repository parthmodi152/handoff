package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPublishWithSubscriber(t *testing.T) {
	b := NewBroker()
	ch, unsub := b.Subscribe("req-1")
	defer unsub()

	event := Event{
		RequestID: "req-1",
		LoopID:    "loop-1",
		Status:    "completed",
		Data:      json.RawMessage(`{"status":"completed"}`),
	}
	b.Publish(event)

	select {
	case got := <-ch:
		if got.RequestID != "req-1" {
			t.Errorf("expected req-1, got %s", got.RequestID)
		}
		if got.Status != "completed" {
			t.Errorf("expected completed, got %s", got.Status)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestPublishWithoutSubscriber(t *testing.T) {
	b := NewBroker()
	// Should not panic or block
	b.Publish(Event{RequestID: "req-1", Status: "completed"})
}

func TestUnsubscribe(t *testing.T) {
	b := NewBroker()
	_, unsub := b.Subscribe("req-1")
	unsub()

	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.subs["req-1"]) != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", len(b.subs["req-1"]))
	}
	if _, exists := b.subs["req-1"]; exists {
		t.Error("expected request key to be cleaned up")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	b := NewBroker()
	ch1, unsub1 := b.Subscribe("req-1")
	defer unsub1()
	ch2, unsub2 := b.Subscribe("req-1")
	defer unsub2()

	b.Publish(Event{RequestID: "req-1", Status: "cancelled"})

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Status != "cancelled" {
				t.Errorf("subscriber %d: expected cancelled, got %s", i, got.Status)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out", i)
		}
	}
}

func TestDifferentRequestIDs(t *testing.T) {
	b := NewBroker()
	ch1, unsub1 := b.Subscribe("req-1")
	defer unsub1()
	ch2, unsub2 := b.Subscribe("req-2")
	defer unsub2()

	b.Publish(Event{RequestID: "req-1", Status: "completed"})

	// ch1 should receive
	select {
	case <-ch1:
	case <-time.After(time.Second):
		t.Fatal("ch1 timed out")
	}

	// ch2 should NOT receive
	select {
	case <-ch2:
		t.Fatal("ch2 should not have received event for req-1")
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}
