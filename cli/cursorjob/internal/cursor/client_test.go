package cursor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// newTestServer returns a client whose transport serves handler in-process.
// Nothing binds a socket, so the suite runs anywhere — including sandboxes and
// CI runners without loopback networking.
func newTestServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		recorder := httptest.NewRecorder()
		handler(recorder, req)
		resp := recorder.Result()
		resp.Request = req
		return resp, nil
	})}
	return NewClient("https://api.cursor.test", "crsr_test_key", httpClient)
}

func TestCreateAgentSendsAuthAndReturnsIDs(t *testing.T) {
	var (
		gotPath   string
		gotAuth   string
		gotMethod string
		gotBody   CreateAgentRequest
	)

	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotMethod = r.URL.Path, r.Header.Get("Authorization"), r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"agent": {"id": "bc-1", "status": "ACTIVE", "url": "https://cursor.com/agents/bc-1", "latestRunId": "run-1"},
			"run":   {"id": "run-1", "agentId": "bc-1", "status": "CREATING"}
		}`))
	})

	resp, err := client.CreateAgent(context.Background(), CreateAgentRequest{
		Prompt: Prompt{Text: "fix the flaky test"},
		Repos:  []Repo{{URL: "https://github.com/benfdking/sdsf", StartingRef: "main"}},
	})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/agents" {
		t.Errorf("path = %q, want /v1/agents", gotPath)
	}
	if gotAuth != "Bearer crsr_test_key" {
		t.Errorf("Authorization = %q, want Bearer crsr_test_key", gotAuth)
	}
	if gotBody.Prompt.Text != "fix the flaky test" {
		t.Errorf("prompt.text = %q", gotBody.Prompt.Text)
	}
	if len(gotBody.Repos) != 1 || gotBody.Repos[0].StartingRef != "main" {
		t.Errorf("repos = %+v", gotBody.Repos)
	}
	if resp.Run.ID != "run-1" || resp.Agent.ID != "bc-1" {
		t.Errorf("got agent %q run %q", resp.Agent.ID, resp.Run.ID)
	}
}

// A create response without a run id would otherwise surface later as a
// confusing 404 from the attach path.
func TestCreateAgentRejectsResponseWithoutRunID(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"agent": {"id": "bc-1"}}`))
	})

	if _, err := client.CreateAgent(context.Background(), CreateAgentRequest{}); err == nil {
		t.Fatal("expected an error when the response carries no run id")
	}
}

func TestCreateAgentBackfillsAgentIDOntoRun(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"agent": {"id": "bc-1"}, "run": {"id": "run-1", "status": "CREATING"}}`))
	})

	resp, err := client.CreateAgent(context.Background(), CreateAgentRequest{})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	if resp.Run.AgentID != "bc-1" {
		t.Errorf("run.agentId = %q, want bc-1", resp.Run.AgentID)
	}
}

func TestAPIErrorParsesEnvelopeAndClassifies(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantCode    string
		wantAuth    bool
		wantRetry   bool
		wantExpired bool
	}{
		{
			name: "unauthorized", status: 401, body: `{"code":"unauthorized","message":"Invalid API key"}`,
			wantCode: "unauthorized", wantAuth: true,
		},
		{
			name: "rate limited", status: 429, body: `{"code":"rate_limited","message":"slow down"}`,
			wantCode: "rate_limited", wantRetry: true,
		},
		{
			name: "server error", status: 503, body: `upstream unavailable`,
			wantRetry: true,
		},
		{
			name: "stream expired", status: 410, body: `{"code":"stream_expired","message":"gone"}`,
			wantCode: "stream_expired", wantExpired: true,
		},
		{
			name: "nested envelope", status: 400, body: `{"error":{"code":"bad_request","message":"nope"}}`,
			wantCode: "bad_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})

			_, err := client.GetRun(context.Background(), "bc-1", "run-1")
			apiErr, ok := errors.AsType[*APIError](err)
			if !ok {
				t.Fatalf("error = %v, want *APIError", err)
			}
			if apiErr.StatusCode != tt.status {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.status)
			}
			if apiErr.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", apiErr.Code, tt.wantCode)
			}
			if apiErr.IsAuth() != tt.wantAuth {
				t.Errorf("IsAuth() = %v, want %v", apiErr.IsAuth(), tt.wantAuth)
			}
			if apiErr.IsRetryable() != tt.wantRetry {
				t.Errorf("IsRetryable() = %v, want %v", apiErr.IsRetryable(), tt.wantRetry)
			}
			if apiErr.IsStreamExpired() != tt.wantExpired {
				t.Errorf("IsStreamExpired() = %v, want %v", apiErr.IsStreamExpired(), tt.wantExpired)
			}
		})
	}
}

func TestListAgentsEncodesQueryParameters(t *testing.T) {
	var gotQuery string
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"items": [{"id": "bc-1"}], "nextCursor": "abc"}`))
	})

	includeArchived := false
	page, err := client.ListAgents(context.Background(), ListAgentsOptions{
		Limit:           50,
		Cursor:          "page-2",
		IncludeArchived: &includeArchived,
	})
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}

	want := "cursor=page-2&includeArchived=false&limit=50"
	if gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
	if len(page.Items) != 1 || page.NextCursor != "abc" {
		t.Errorf("page = %+v", page)
	}
}

func TestResolveRunIDUsesLatestRunID(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/agents/bc-1" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id": "bc-1", "latestRunId": "run-9"}`))
	})

	runID, err := client.ResolveRunID(context.Background(), "bc-1", "")
	if err != nil {
		t.Fatalf("ResolveRunID: %v", err)
	}
	if runID != "run-9" {
		t.Errorf("runID = %q, want run-9", runID)
	}
}

// Agents predating latestRunId must still resolve, via the runs listing.
func TestResolveRunIDFallsBackToRunsListing(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/agents/bc-1":
			_, _ = w.Write([]byte(`{"id": "bc-1"}`))
		case "/v1/agents/bc-1/runs":
			_, _ = w.Write([]byte(`{"items": [{"id": "run-3", "status": "RUNNING"}]}`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})

	runID, err := client.ResolveRunID(context.Background(), "bc-1", "")
	if err != nil {
		t.Fatalf("ResolveRunID: %v", err)
	}
	if runID != "run-3" {
		t.Errorf("runID = %q, want run-3", runID)
	}
}

func TestResolveRunIDPassesThroughExplicitID(t *testing.T) {
	client := newTestServer(t, func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %q for an explicit run id", r.URL.Path)
	})

	runID, err := client.ResolveRunID(context.Background(), "bc-1", "run-explicit")
	if err != nil {
		t.Fatalf("ResolveRunID: %v", err)
	}
	if runID != "run-explicit" {
		t.Errorf("runID = %q", runID)
	}
}

func TestIsTerminalTreatsUnknownStatusAsInFlight(t *testing.T) {
	terminal := []string{RunStatusFinished, RunStatusError, RunStatusCancelled, RunStatusExpired}
	for _, status := range terminal {
		if !IsTerminal(status) {
			t.Errorf("IsTerminal(%q) = false, want true", status)
		}
	}
	// An unrecognised status must not end a wait — that would report success
	// for a run that is still going.
	for _, status := range []string{RunStatusCreating, RunStatusRunning, "QUEUED", ""} {
		if IsTerminal(status) {
			t.Errorf("IsTerminal(%q) = true, want false", status)
		}
	}
}

func TestCancelRunPostsToCancelPath(t *testing.T) {
	var gotPath, gotMethod string
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusNoContent)
	})

	if err := client.CancelRun(context.Background(), "bc-1", "run-1"); err != nil {
		t.Fatalf("CancelRun: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/agents/bc-1/runs/run-1/cancel" {
		t.Errorf("path = %q", gotPath)
	}
}
