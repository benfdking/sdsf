package cursorautomations

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// newTestClient serves responses in-process, so the suite needs no network.
func newTestClient(t *testing.T, status int, body string, inspect func(*http.Request)) *Client {
	t.Helper()
	client := NewClient("session-token", "42")
	client.baseURL = "https://cursor.test/api/automations"
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if inspect != nil {
			inspect(req)
		}
		return &http.Response{
			StatusCode: status,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}
	return client
}

func TestCreateSendsSessionCookieAndReturnsID(t *testing.T) {
	var (
		gotCookie string
		gotOrigin string
		gotPath   string
		gotBody   createRequest
	)

	client := newTestClient(t, http.StatusOK,
		`{"workflow":{"workflow":{"automationId":"auto-9"}}}`,
		func(req *http.Request) {
			gotCookie = req.Header.Get("Cookie")
			gotOrigin = req.Header.Get("Origin")
			gotPath = req.URL.Path
			if err := json.NewDecoder(req.Body).Decode(&gotBody); err != nil {
				t.Errorf("decode request: %v", err)
			}
		})

	automation, err := NewAutomation(validConfig(), "prompt")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}

	id, err := client.Create(context.Background(), automation)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id != "auto-9" {
		t.Errorf("id = %q, want auto-9", id)
	}
	if !strings.Contains(gotCookie, "WorkosCursorSessionToken=session-token") {
		t.Errorf("Cookie = %q, want the session token", gotCookie)
	}
	if !strings.Contains(gotCookie, "team_id=42") {
		t.Errorf("Cookie = %q, want the team id", gotCookie)
	}
	if gotOrigin != "https://cursor.com" {
		t.Errorf("Origin = %q", gotOrigin)
	}
	if !strings.HasSuffix(gotPath, "/create-automation") {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody.TeamID != 42 {
		t.Errorf("teamId = %d, want 42", gotBody.TeamID)
	}
}

// The id has been seen at three different depths; all must work.
func TestCreateFindsAutomationIDAtAnyDepth(t *testing.T) {
	tests := map[string]string{
		"nested twice": `{"workflow":{"workflow":{"automationId":"deep"}}}`,
		"nested once":  `{"workflow":{"automationId":"deep"}}`,
		"top level":    `{"automationId":"deep"}`,
	}

	automation, err := NewAutomation(validConfig(), "prompt")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			client := newTestClient(t, http.StatusOK, body, nil)
			id, err := client.Create(context.Background(), automation)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if id != "deep" {
				t.Errorf("id = %q, want deep", id)
			}
		})
	}
}

// A create that reports success but returns no id would otherwise leave state
// pointing at nothing.
func TestCreateRejectsResponseWithoutID(t *testing.T) {
	client := newTestClient(t, http.StatusOK, `{"workflow":{}}`, nil)
	automation, err := NewAutomation(validConfig(), "prompt")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}

	if _, err := client.Create(context.Background(), automation); err == nil {
		t.Fatal("expected an error when no automationId comes back")
	}
}

func TestCreateRejectsNonNumericTeamID(t *testing.T) {
	client := newTestClient(t, http.StatusOK, `{}`, func(*http.Request) {
		t.Error("no request should be sent for an invalid team id")
	})
	client.teamID = "not-a-number"

	automation, err := NewAutomation(validConfig(), "prompt")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}
	if _, err := client.Create(context.Background(), automation); err == nil {
		t.Fatal("expected an error for a non-numeric team id")
	}
}

func TestUpdateRequiresAutomationID(t *testing.T) {
	client := newTestClient(t, http.StatusOK, `{}`, func(*http.Request) {
		t.Error("no request should be sent without an automation id")
	})

	automation, err := NewAutomation(validConfig(), "prompt")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}
	if err := client.Update(context.Background(), automation, ""); err == nil {
		t.Fatal("expected an error for an empty automation id")
	}
}

func TestUpdatePostsToUpdateEndpoint(t *testing.T) {
	var gotPath string
	var gotBody updateRequest
	client := newTestClient(t, http.StatusOK, `{}`, func(req *http.Request) {
		gotPath = req.URL.Path
		_ = json.NewDecoder(req.Body).Decode(&gotBody)
	})

	automation, err := NewAutomation(validConfig(), "prompt")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}
	if err := client.Update(context.Background(), automation, "auto-3"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !strings.HasSuffix(gotPath, "/update-automation") {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody.AutomationID != "auto-3" {
		t.Errorf("automationId = %q", gotBody.AutomationID)
	}
}

func TestAPIErrorClassification(t *testing.T) {
	tests := map[string]struct {
		status              int
		body                string
		wantAuth            bool
		wantSlackDisconnect bool
	}{
		"expired session": {http.StatusUnauthorized, "unauthorized", true, false},
		"forbidden":       {http.StatusForbidden, "forbidden", true, false},
		"slack missing": {
			http.StatusBadRequest,
			`{"error":"Please connect your Slack account to continue"}`,
			false, true,
		},
		"other bad request": {http.StatusBadRequest, "malformed", false, false},
		"server error":      {http.StatusInternalServerError, "boom", false, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			client := newTestClient(t, tt.status, tt.body, nil)

			_, err := client.List(context.Background())
			apiErr, ok := errors.AsType[*APIError](err)
			if !ok {
				t.Fatalf("error = %v, want *APIError", err)
			}
			if apiErr.StatusCode != tt.status {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.status)
			}
			if apiErr.IsAuth() != tt.wantAuth {
				t.Errorf("IsAuth() = %v, want %v", apiErr.IsAuth(), tt.wantAuth)
			}
			if apiErr.IsSlackNotConnected() != tt.wantSlackDisconnect {
				t.Errorf("IsSlackNotConnected() = %v, want %v", apiErr.IsSlackNotConnected(), tt.wantSlackDisconnect)
			}
		})
	}
}

func TestTruncateStepsBackToRuneBoundary(t *testing.T) {
	// "é" is two bytes; cutting at 3 would split it.
	if got := truncate("aaé", 3); !strings.HasSuffix(got, "…") || strings.Contains(got, "�") {
		t.Errorf("truncate produced %q, want a clean cut", got)
	}
	if got := truncate("short", 100); got != "short" {
		t.Errorf("truncate(%q) = %q, want it unchanged", "short", got)
	}
}
