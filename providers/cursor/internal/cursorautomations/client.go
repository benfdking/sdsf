package cursorautomations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const defaultBaseURL = "https://cursor.com/api/automations"

// Client calls Cursor's (unofficial) automations API. Auth is the
// WorkosCursorSessionToken web session cookie plus the team id. There is no
// public API, so this hits the same endpoints the dashboard frontend uses.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	sessionToken string
	teamID       string
}

// NewClient builds a client from a session token and team id.
func NewClient(sessionToken, teamID string) *Client {
	return &Client{
		baseURL:      defaultBaseURL,
		sessionToken: sessionToken,
		teamID:       teamID,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is a non-2xx response, carrying enough to classify the failure
// (expired token vs contract drift vs transient) at the call site.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("cursor API returned status %d: %s", e.StatusCode, e.Body)
}

// IsAuth reports a likely-expired/invalid session token (rotate it).
func (e *APIError) IsAuth() bool { return e.StatusCode == 401 || e.StatusCode == 403 }

// IsSlackNotConnected reports the specific 400 Cursor returns when the acting
// account has no Slack workspace connected, so it can't resolve the channels an
// automation references (trigger, notified channels, slack / readSlack actions).
// This fires even when channels are given as IDs, so the fix is to connect Slack
// in Cursor as the token owner — not to change the config. Matched on the message
// substring (case-insensitively) because the API has no stable error code for it,
// so it's already brittle to any wording change on their side.
func (e *APIError) IsSlackNotConnected() bool {
	return e.StatusCode == 400 && strings.Contains(strings.ToLower(e.Body), "connect your slack account")
}

// Update pushes an automation's full config to Cursor via /update-automation.
func (c *Client) Update(ctx context.Context, a Automation, automationID string) error {
	if automationID == "" {
		return fmt.Errorf("%s: update requires a non-empty automationId", a.Dir)
	}
	_, err := c.post(ctx, "/update-automation", a.toUpdateRequest(automationID))
	return err
}

// List fetches all automations the session can see.
func (c *Client) List(ctx context.Context) (json.RawMessage, error) {
	return c.post(ctx, "/list-automations", map[string]any{})
}

// Create makes a new automation via /create-automation and returns its
// server-assigned id. The create request carries the full desired config, so no
// follow-up Update is needed (and issuing one would be a duplicate write).
func (c *Client) Create(ctx context.Context, a Automation) (string, error) {
	teamID, err := strconv.ParseInt(c.teamID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid team id %q: %w", c.teamID, err)
	}

	raw, err := c.post(ctx, "/create-automation", a.toCreateRequest(teamID))
	if err != nil {
		return "", err
	}

	id, err := parseCreatedID(raw)
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("create succeeded but no automationId in response: %s", truncate(string(raw), 512))
	}
	return id, nil
}

func parseCreatedID(raw []byte) (string, error) {
	var resp struct {
		AutomationID string `json:"automationId"`
		Workflow     struct {
			AutomationID string `json:"automationId"`
			Workflow     struct {
				AutomationID string `json:"automationId"`
			} `json:"workflow"`
		} `json:"workflow"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("parsing create response: %w", err)
	}
	// Deepest path is the one observed live; the shallower two are documented fallbacks.
	for _, id := range []string{resp.Workflow.Workflow.AutomationID, resp.Workflow.AutomationID, resp.AutomationID} {
		if id != "" {
			return id, nil
		}
	}
	return "", nil
}

func (c *Client) post(ctx context.Context, path string, payload any) (json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://cursor.com")
	req.Header.Set("Cookie", fmt.Sprintf("team_id=%s; WorkosCursorSessionToken=%s", c.teamID, c.sessionToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling cursor API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1024)}
	}

	return respBody, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Back up to a rune boundary so a multi-byte character straddling the limit
	// doesn't render as mojibake in error and diff output.
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
