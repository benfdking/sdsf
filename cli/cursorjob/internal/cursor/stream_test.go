package cursor

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func collect(t *testing.T, body string) []Event {
	t.Helper()
	var events []Event
	if err := scanEvents(strings.NewReader(body), func(e Event) error {
		events = append(events, e)
		return nil
	}); err != nil {
		t.Fatalf("scanEvents: %v", err)
	}
	return events
}

func TestScanEventsParsesTypesAndData(t *testing.T) {
	events := collect(t, "event: status\ndata: {\"status\":\"RUNNING\"}\n\nevent: assistant\ndata: {\"text\":\"hi\"}\n\n")

	want := []Event{
		{Type: "status", Data: []byte(`{"status":"RUNNING"}`)},
		{Type: "assistant", Data: []byte(`{"text":"hi"}`)},
	}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("events = %+v, want %+v", events, want)
	}
}

// A JSON payload split across data lines must be rejoined with newlines, per
// the SSE spec, or it will not parse.
func TestScanEventsJoinsMultiLineData(t *testing.T) {
	events := collect(t, "event: result\ndata: {\ndata: \"status\":\"FINISHED\"\ndata: }\n\n")

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	var payload ResultPayload
	if err := events[0].Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != RunStatusFinished {
		t.Errorf("status = %q, want FINISHED", payload.Status)
	}
}

// The id field persists across subsequent events, which is what makes
// Last-Event-ID resumption correct.
func TestScanEventsCarriesIDForward(t *testing.T) {
	events := collect(t, "id: 7\nevent: assistant\ndata: a\n\nevent: assistant\ndata: b\n\n")

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].ID != "7" || events[1].ID != "7" {
		t.Errorf("ids = %q, %q; want both 7", events[0].ID, events[1].ID)
	}
}

func TestScanEventsIgnoresCommentsAndHandlesCRLF(t *testing.T) {
	events := collect(t, ": keepalive\r\nevent: heartbeat\r\ndata: {}\r\n\r\n")

	if len(events) != 1 || events[0].Type != EventHeartbeat {
		t.Fatalf("events = %+v, want a single heartbeat", events)
	}
	if string(events[0].Data) != "{}" {
		t.Errorf("data = %q, want {}", events[0].Data)
	}
}

// A stream cut off before its trailing blank line still holds a usable event.
func TestScanEventsDispatchesTrailingEvent(t *testing.T) {
	events := collect(t, "event: done\ndata: {}")

	if len(events) != 1 || events[0].Type != EventDone {
		t.Errorf("events = %+v, want a single done event", events)
	}
}

func TestScanEventsHandlesDataWithoutSpaceAndBareField(t *testing.T) {
	events := collect(t, "event:assistant\ndata:{\"text\":\"x\"}\n\n")

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type != EventAssistant || string(events[0].Data) != `{"text":"x"}` {
		t.Errorf("event = %+v", events[0])
	}
}

func TestStreamRunSetsHeadersAndReportsLastEventID(t *testing.T) {
	var gotAccept, gotLastEventID, gotPath string

	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotLastEventID = r.Header.Get("Last-Event-ID")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("id: 41\nevent: assistant\ndata: {\"text\":\"one\"}\n\nid: 42\nevent: done\ndata: {}\n\n"))
	})

	var types []string
	lastID, err := client.StreamRun(context.Background(), "bc-1", "run-1", "40", func(e Event) error {
		types = append(types, e.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamRun: %v", err)
	}

	if gotPath != "/v1/agents/bc-1/runs/run-1/stream" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAccept != "text/event-stream" {
		t.Errorf("Accept = %q", gotAccept)
	}
	if gotLastEventID != "40" {
		t.Errorf("Last-Event-ID = %q, want 40", gotLastEventID)
	}
	if lastID != "42" {
		t.Errorf("lastEventID = %q, want 42", lastID)
	}
	if !reflect.DeepEqual(types, []string{EventAssistant, EventDone}) {
		t.Errorf("types = %v", types)
	}
}

// A non-2xx on the stream endpoint has to arrive as an APIError so WatchRun can
// tell "retention expired" from "your key is wrong".
func TestStreamRunReturnsAPIErrorOnFailure(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`{"code":"stream_expired","message":"gone"}`))
	})

	_, err := client.StreamRun(context.Background(), "bc-1", "run-1", "", func(Event) error { return nil })
	apiErr, ok := errType(err)
	if !ok {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if !apiErr.IsStreamExpired() {
		t.Errorf("IsStreamExpired() = false for %v", apiErr)
	}
}

// StreamRun must return the id of the last event the callback actually saw, so
// a resumed stream doesn't replay or skip.
func TestStreamRunReportsLastIDWhenCallbackStops(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("id: 1\nevent: assistant\ndata: a\n\nid: 2\nevent: assistant\ndata: b\n\n"))
	})

	stop := errStreamDone
	lastID, err := client.StreamRun(context.Background(), "bc-1", "run-1", "", func(e Event) error {
		if e.ID == "1" {
			return stop
		}
		return nil
	})
	if err != stop {
		t.Fatalf("err = %v, want the callback's error", err)
	}
	if lastID != "1" {
		t.Errorf("lastEventID = %q, want 1", lastID)
	}
}

func errType(err error) (*APIError, bool) {
	apiErr, ok := err.(*APIError)
	return apiErr, ok
}
