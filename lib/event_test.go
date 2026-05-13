package lib

import (
	"encoding/json"
	"testing"
)

func TestEventsTracker_Add(t *testing.T) {
	tracker := &EventsTracker{}
	event := &Event{EventType: "test"}

	err := tracker.Add(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tracker.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(tracker.Events))
	}
}

func TestEventsTracker_Add_Nil(t *testing.T) {
	var tracker *EventsTracker

	err := tracker.Add(&Event{})
	if err == nil {
		t.Error("expected error for nil tracker")
	}
}

func TestEventsTracker_Refer(t *testing.T) {
	tracker := &EventsTracker{}
	ref := "test-reference"

	tracker.Refer(ref)

	if tracker.Reference != ref {
		t.Errorf("expected reference %s, got %s", ref, tracker.Reference)
	}
}

func TestEventsTracker_GetReference(t *testing.T) {
	tracker := &EventsTracker{Reference: "test-ref"}

	ref := tracker.GetReference()
	if ref != "test-ref" {
		t.Errorf("expected 'test-ref', got %s", ref)
	}
}

func TestEventsTracker_Reset(t *testing.T) {
	tracker := &EventsTracker{
		Reference: "test",
		Events:    Events{&Event{EventType: "test"}},
	}

	events := tracker.Reset()

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if tracker.Reference != "" {
		t.Errorf("expected empty reference after reset, got %s", tracker.Reference)
	}
	if len(tracker.Events) != 0 {
		t.Errorf("expected empty events after reset, got %d", len(tracker.Events))
	}
}

func TestEvents_Len(t *testing.T) {
	events := Events{&Event{}, &Event{}}

	if events.Len() != 2 {
		t.Errorf("expected length 2, got %d", events.Len())
	}
}

func TestEvent_MarshalJSON(t *testing.T) {
	event := &Event{
		EventType: "test",
		Height:    100,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}
}

func TestEvent_UnmarshalJSON(t *testing.T) {
	jsonData := `{"eventType":"test","height":100}`

	var event Event
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.EventType != "test" {
		t.Errorf("expected eventType 'test', got %s", event.EventType)
	}
	if event.Height != 100 {
		t.Errorf("expected height 100, got %d", event.Height)
	}
}

func TestEventOrderBookSwap_MarshalJSON(t *testing.T) {
	swap := &EventOrderBookSwap{
		SoldAmount:   1000,
		BoughtAmount: 2000,
	}

	data, err := json.Marshal(swap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}
}

func TestEventOrderBookSwap_UnmarshalJSON(t *testing.T) {
	jsonData := `{"soldAmount":1000,"boughtAmount":2000}`

	var swap EventOrderBookSwap
	err := json.Unmarshal([]byte(jsonData), &swap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if swap.SoldAmount != 1000 {
		t.Errorf("expected soldAmount 1000, got %d", swap.SoldAmount)
	}
	if swap.BoughtAmount != 2000 {
		t.Errorf("expected boughtAmount 2000, got %d", swap.BoughtAmount)
	}
}
