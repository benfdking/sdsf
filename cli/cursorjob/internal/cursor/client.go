// Package cursor is a client for Cursor's Cloud Agents REST API (api.cursor.com/v1).
//
// The API models work as agents that own an ordered list of runs: creating an
// agent starts its first run, and follow-up prompts append further runs. Run
// status is what callers actually wait on, so most of this package is shaped
// around a run reaching a terminal state.
package cursor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultBaseURL is the public API host. Override it in tests, or via
// CURSOR_API_BASE_URL, so no test ever depends on reaching the network.
const DefaultBaseURL = "https://api.cursor.com"

// Client talks to the Cloud Agents API. The zero value is not usable; build one
// with NewClient.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient builds a client. An empty baseURL or nil httpClient fall back to
// sensible defaults. The timeout is deliberately left off the shared client:
// the SSE stream endpoint is long-lived, and a client-wide timeout would cut it
// off mid-run. Per-request deadlines come from the context instead.
func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// APIError is a non-2xx response. Code and Message come from Cursor's error
// envelope ({"code":..., "message":...}) when it parses, so callers can branch
// on a stable code rather than scraping prose.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	switch {
	case e.Code != "" && e.Message != "":
		return fmt.Sprintf("cursor API %d (%s): %s", e.StatusCode, e.Code, e.Message)
	case e.Message != "":
		return fmt.Sprintf("cursor API %d: %s", e.StatusCode, e.Message)
	default:
		return fmt.Sprintf("cursor API %d: %s", e.StatusCode, e.Body)
	}
}

// IsAuth reports a missing, invalid, or under-scoped API key.
func (e *APIError) IsAuth() bool { return e.StatusCode == 401 || e.StatusCode == 403 }

// IsNotFound reports an unknown agent or run id.
func (e *APIError) IsNotFound() bool { return e.StatusCode == 404 }

// IsRateLimited reports that the per-team per-minute quota is exhausted.
func (e *APIError) IsRateLimited() bool { return e.StatusCode == 429 }

// IsStreamExpired reports that a run's SSE event buffer has aged out. The run
// itself is unaffected — callers should switch to polling GetRun.
func (e *APIError) IsStreamExpired() bool {
	return e.StatusCode == http.StatusGone || e.Code == "stream_expired"
}

// IsRetryable reports a failure that is plausibly transient, so a caller in a
// reconnect loop should back off rather than give up.
func (e *APIError) IsRetryable() bool {
	return e.StatusCode == 429 || e.StatusCode >= 500
}

func newAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode, Body: truncate(strings.TrimSpace(string(body)), 1024)}
	var envelope struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		apiErr.Code = firstNonEmpty(envelope.Code, envelope.Error.Code)
		apiErr.Message = firstNonEmpty(envelope.Message, envelope.Error.Message)
	}
	return apiErr
}

// newRequest builds an authenticated request. Cursor accepts the API key as a
// bearer token on the Cloud Agents API, which is the only surface this uses.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// do issues a request and decodes a JSON response into result, which may be nil
// when the body is not needed.
func (c *Client) do(ctx context.Context, method, path string, body, result any) error {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call cursor API: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read cursor response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp.StatusCode, responseBody)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(responseBody, result); err != nil {
		return fmt.Errorf("decode cursor response: %w", err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
