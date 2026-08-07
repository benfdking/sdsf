package cursor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// Stream event types emitted by GET /v1/agents/{id}/runs/{runId}/stream.
const (
	EventStatus            = "status"
	EventAssistant         = "assistant"
	EventThinking          = "thinking"
	EventToolCall          = "tool_call"
	EventInteractionUpdate = "interaction_update"
	EventHeartbeat         = "heartbeat"
	EventResult            = "result"
	EventError             = "error"
	EventDone              = "done"
)

// Event is a single server-sent event. Data is the raw payload, left undecoded
// so that unknown event types still pass through intact.
type Event struct {
	ID   string
	Type string
	Data []byte
}

// StatusPayload is the `status` event, sent first and again after any reconnect.
type StatusPayload struct {
	RunID  string `json:"runId"`
	Status string `json:"status"`
}

// TextPayload is the `assistant` and `thinking` events, which carry deltas
// rather than whole messages.
type TextPayload struct {
	Text string `json:"text"`
}

// ToolCallPayload is the `tool_call` event. Status is "running" or "completed".
type ToolCallPayload struct {
	CallID string          `json:"callId"`
	Name   string          `json:"name"`
	Status string          `json:"status"`
	Args   json.RawMessage `json:"args,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}

// ResultPayload is the terminal `result` event carrying the final state.
type ResultPayload struct {
	RunID      string `json:"runId"`
	Status     string `json:"status"`
	Text       string `json:"text,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Git        *Git   `json:"git,omitempty"`
}

// ErrorPayload is the `error` event.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Decode unmarshals an event payload into target.
func (e Event) Decode(target any) error {
	if len(bytes.TrimSpace(e.Data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(e.Data, target); err != nil {
		return fmt.Errorf("decode %q event: %w", e.Type, err)
	}
	return nil
}

// scanEvents reads an SSE body and calls fn for each complete event. It follows
// the WHATWG event-stream parse rules closely enough for this API: `field: value`
// lines accumulate until a blank line dispatches, lines opening with a colon are
// comments, and repeated data lines join with newlines.
func scanEvents(r io.Reader, fn func(Event) error) error {
	scanner := bufio.NewScanner(r)
	// Tool call args and results can be large; the 64 KiB default token size is
	// not enough and would abort an otherwise healthy stream.
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	var (
		eventType string
		eventID   string
		data      []string
	)

	dispatch := func() error {
		if eventType == "" && len(data) == 0 {
			return nil
		}
		event := Event{
			ID:   eventID,
			Type: eventType,
			Data: []byte(strings.Join(data, "\n")),
		}
		eventType = ""
		data = nil
		return fn(event)
	}

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		switch {
		case line == "":
			if err := dispatch(); err != nil {
				return err
			}
		case strings.HasPrefix(line, ":"):
			// Comment / keepalive padding.
		default:
			field, value, found := strings.Cut(line, ":")
			if !found {
				field, value = line, ""
			}
			value = strings.TrimPrefix(value, " ")
			switch field {
			case "event":
				eventType = value
			case "data":
				data = append(data, value)
			case "id":
				// The id field persists across events per the SSE spec, so it is
				// intentionally not reset by dispatch.
				eventID = value
			case "retry":
				// Reconnection backoff is handled by WatchRun instead.
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	// A stream that ends mid-event still has a final event worth delivering.
	return dispatch()
}

// StreamRun opens the SSE stream for a run and calls fn for every event until
// the stream ends, fn returns an error, or ctx is cancelled.
//
// lastEventID resumes an interrupted stream from the last event the caller saw;
// pass "" to start from the beginning of the retained buffer. The returned
// string is the most recent event id observed, suitable for the next attempt.
//
// Reaching the end of the stream is not an error here — deciding whether the
// run actually finished is WatchRun's job.
func (c *Client) StreamRun(ctx context.Context, agentID, runID, lastEventID string, fn func(Event) error) (string, error) {
	path := "/v1/agents/" + url.PathEscape(agentID) + "/runs/" + url.PathEscape(runID) + "/stream"
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return lastEventID, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return lastEventID, fmt.Errorf("open run stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		return lastEventID, newAPIError(resp.StatusCode, body)
	}

	seen := lastEventID
	err = scanEvents(resp.Body, func(event Event) error {
		if event.ID != "" {
			seen = event.ID
		}
		return fn(event)
	})
	return seen, err
}
