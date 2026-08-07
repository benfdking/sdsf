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

// baseURL is the automations endpoint the Cursor dashboard calls.
const baseURL = "https://cursor.com/api/automations"

// requestTimeout bounds a single call. Every operation here is a small,
// interactive POST, so a caller stuck behind a hung connection is a bug rather
// than a slow but healthy request.
const requestTimeout = 30 * time.Second

// API is the set of operations a caller needs to reconcile automations. It
// exists so callers can substitute a fake in tests without an HTTP round trip.
type API interface {
	List(ctx context.Context) (json.RawMessage, error)
	Create(ctx context.Context, automation Automation) (string, error)
	Update(ctx context.Context, automation Automation, automationID string) error
}

// Client calls the automations API as a signed-in dashboard user.
//
// There is no API-key authentication for this surface: requests carry the
// WorkosCursorSessionToken browser cookie and a team id, exactly as the
// dashboard sends them.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	sessionToken string
	teamID       string
}

// Ensure the concrete client satisfies the interface callers depend on.
var _ API = (*Client)(nil)

// NewClient builds a client for one team's automations.
func NewClient(sessionToken, teamID string) *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: requestTimeout},
		baseURL:      baseURL,
		sessionToken: sessionToken,
		teamID:       teamID,
	}
}

// APIError is a non-2xx response. The status and body are both kept so a caller
// can tell an expired session from a changed contract.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("cursor API returned status %d: %s", e.StatusCode, e.Body)
}

// IsAuth reports a session token that is missing, expired, or not permitted to
// act on the team. The fix is to supply a fresh one.
func (e *APIError) IsAuth() bool {
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

// IsSlackNotConnected reports the specific rejection Cursor returns when the
// account behind the session token has no Slack workspace connected, so it
// cannot resolve the channels an automation names.
//
// This fires even when channels are given as ids, so the fix is to connect
// Slack in Cursor as the token's owner rather than to change the automation.
// There is no stable error code for it, so this matches on the message and is
// therefore brittle to any rewording upstream.
func (e *APIError) IsSlackNotConnected() bool {
	return e.StatusCode == http.StatusBadRequest &&
		strings.Contains(strings.ToLower(e.Body), "connect your slack account")
}

// List returns the raw /list-automations response. It stays raw because
// callers project different subsets of a listed automation, and the endpoint
// returns more than this package models.
func (c *Client) List(ctx context.Context) (json.RawMessage, error) {
	return c.post(ctx, "/list-automations", struct{}{})
}

// Create creates an automation and returns the id the server assigned. The
// request carries the full desired configuration, so no follow-up Update is
// needed and issuing one would just repeat the write.
func (c *Client) Create(ctx context.Context, automation Automation) (string, error) {
	teamID, err := strconv.ParseInt(c.teamID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid team id %q: %w", c.teamID, err)
	}

	raw, err := c.post(ctx, "/create-automation", automation.toCreateRequest(teamID))
	if err != nil {
		return "", err
	}

	id, err := parseCreatedID(raw)
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("create succeeded but returned no automationId: %s", truncate(string(raw), 512))
	}
	return id, nil
}

// Update replaces an existing automation's configuration.
func (c *Client) Update(ctx context.Context, automation Automation, automationID string) error {
	if automationID == "" {
		return fmt.Errorf("update %q: automationId must not be empty", automation.Config.Name)
	}
	_, err := c.post(ctx, "/update-automation", automation.toUpdateRequest(automationID))
	return err
}

// parseCreatedID pulls the new automation's id out of a create response.
//
// The id has been observed nested two levels deep, but the shallower positions
// are checked too so a response that flattens does not break creation.
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

	candidates := []string{
		resp.Workflow.Workflow.AutomationID,
		resp.Workflow.AutomationID,
		resp.AutomationID,
	}
	for _, id := range candidates {
		if id != "" {
			return id, nil
		}
	}
	return "", nil
}

// post sends a JSON body to an automations endpoint and returns the raw
// response.
func (c *Client) post(ctx context.Context, path string, payload any) (json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
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

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading cursor response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       truncate(strings.TrimSpace(string(responseBody)), 1024),
		}
	}
	return responseBody, nil
}

// truncate caps a string for use in an error message, stepping back to a rune
// boundary so a multi-byte character split by the limit does not render as
// broken output.
func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
