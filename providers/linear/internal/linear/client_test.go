package linear

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

func newTestClient(t *testing.T, authorization, response string, check func(*http.Request)) *Client {
	t.Helper()
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if check != nil {
			check(req)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(response)),
			Request:    req,
		}, nil
	})}
	return NewClient("https://linear.test/graphql", authorization, httpClient)
}

func TestDoSendsAuthorizationAndDecodesData(t *testing.T) {
	client := newTestClient(t, "lin_api_test", `{"data":{"team":{"id":"team-id","name":"Engineering","key":"ENG","description":null}}}`, func(r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "lin_api_test" {
			t.Errorf("Authorization = %q, want lin_api_test", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var request graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Variables["id"] != "team-id" {
			t.Errorf("id variable = %#v, want team-id", request.Variables["id"])
		}
	})
	team, err := client.Team(context.Background(), "team-id")
	if err != nil {
		t.Fatalf("Team returned error: %v", err)
	}
	if team.ID != "team-id" || team.Name != "Engineering" || team.Key != "ENG" {
		t.Fatalf("unexpected team: %#v", team)
	}
}

func TestDoRecognizesGraphQLNotFound(t *testing.T) {
	client := newTestClient(t, "lin_api_test", `{"errors":[{"message":"Entity not found","extensions":{"code":"ENTITY_NOT_FOUND"}}]}`, nil)
	_, err := client.IssueLabel(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestCreateCustomViewUsesGraphQLInput(t *testing.T) {
	client := newTestClient(t, "lin_api_test", `{"data":{"customViewCreate":{"success":true,"customView":{"id":"view-id","name":"Security","description":null,"color":null,"icon":null,"shared":true,"slugId":"security","filterData":{},"team":{"id":"team-id"}}}}}`, func(r *http.Request) {
		var request graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		input := request.Variables["input"].(map[string]any)
		if input["name"] != "Security" || input["teamId"] != "team-id" {
			t.Errorf("unexpected input: %#v", input)
		}
	})
	view, err := client.CreateCustomView(context.Background(), map[string]any{"name": "Security", "teamId": "team-id"})
	if err != nil {
		t.Fatalf("CreateCustomView returned error: %v", err)
	}
	if view.ID != "view-id" || view.Team == nil || view.Team.ID != "team-id" {
		t.Fatalf("unexpected view: %#v", view)
	}
}

func TestProjectBySlugID(t *testing.T) {
	client := newTestClient(t, "lin_api_test", `{"data":{"projects":{"nodes":[{"id":"project-id","name":"Launch","slugId":"launch-abc123","description":"Ship it","url":"https://linear.app/acme/project/launch-abc123","progress":0.5,"startDate":"2026-07-01","targetDate":null,"status":{"id":"status-id","name":"In Progress","type":"started","color":"#5E6AD2"},"lead":{"id":"user-id"},"teams":{"nodes":[{"id":"team-id"}]}}]}}}`, func(r *http.Request) {
		var request graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		filter := request.Variables["filter"].(map[string]any)
		slug := filter["slugId"].(map[string]any)
		if slug["eq"] != "launch-abc123" {
			t.Errorf("unexpected filter: %#v", filter)
		}
	})

	project, err := client.ProjectBySlugID(context.Background(), "launch-abc123")
	if err != nil {
		t.Fatalf("ProjectBySlugID returned error: %v", err)
	}
	if project.ID != "project-id" || project.Status == nil || project.Status.Name != "In Progress" || len(project.Teams.Nodes) != 1 {
		t.Fatalf("unexpected project: %#v", project)
	}
}
