package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const DefaultEndpoint = "https://api.linear.app/graphql"

var ErrNotFound = errors.New("linear object not found")

type Client struct {
	endpoint      string
	authorization string
	httpClient    *http.Client
}

func NewClient(endpoint, authorization string, httpClient *http.Client) *Client {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{endpoint: endpoint, authorization: authorization, httpClient: httpClient}
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLError struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

func (c *Client) Do(ctx context.Context, query string, variables map[string]any, result any) error {
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("encode GraphQL request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Linear request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authorization)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call Linear API: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read Linear response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Linear API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var envelope graphQLResponse
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("decode Linear response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		messages := make([]string, 0, len(envelope.Errors))
		for _, graphErr := range envelope.Errors {
			messages = append(messages, graphErr.Message)
			code, _ := graphErr.Extensions["code"].(string)
			if code == "ENTITY_NOT_FOUND" || strings.Contains(strings.ToLower(graphErr.Message), "not found") {
				return fmt.Errorf("%w: %s", ErrNotFound, graphErr.Message)
			}
		}
		return fmt.Errorf("Linear GraphQL error: %s", strings.Join(messages, "; "))
	}
	if result != nil {
		if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
			return errors.New("Linear API returned no data")
		}
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return fmt.Errorf("decode Linear data: %w", err)
		}
	}
	return nil
}

type Team struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Key         string  `json:"key"`
	Description *string `json:"description"`
}

func (c *Client) Team(ctx context.Context, id string) (*Team, error) {
	var result struct {
		Team *Team `json:"team"`
	}
	err := c.Do(ctx, `query Team($id: String!) { team(id: $id) { id name key description } }`, map[string]any{"id": id}, &result)
	if err != nil {
		return nil, err
	}
	if result.Team == nil {
		return nil, ErrNotFound
	}
	return result.Team, nil
}

func (c *Client) TeamByKey(ctx context.Context, key string) (*Team, error) {
	var result struct {
		Teams struct {
			Nodes []Team `json:"nodes"`
		} `json:"teams"`
	}
	variables := map[string]any{"filter": map[string]any{"key": map[string]any{"eq": key}}}
	err := c.Do(ctx, `query TeamByKey($filter: TeamFilter!) { teams(first: 2, filter: $filter) { nodes { id name key description } } }`, variables, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Teams.Nodes) == 0 {
		return nil, ErrNotFound
	}
	if len(result.Teams.Nodes) > 1 {
		return nil, fmt.Errorf("more than one Linear team has key %q", key)
	}
	return &result.Teams.Nodes[0], nil
}

type Project struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	SlugID      string  `json:"slugId"`
	Description *string `json:"description"`
	URL         string  `json:"url"`
	Progress    float64 `json:"progress"`
	StartDate   *string `json:"startDate"`
	TargetDate  *string `json:"targetDate"`
	Status      *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Type  string `json:"type"`
		Color string `json:"color"`
	} `json:"status"`
	Lead *struct {
		ID string `json:"id"`
	} `json:"lead"`
	Teams struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	} `json:"teams"`
}

const projectFields = `id name slugId description url progress startDate targetDate status { id name type color } lead { id } teams { nodes { id } }`

func (c *Client) Project(ctx context.Context, id string) (*Project, error) {
	var result struct {
		Project *Project `json:"project"`
	}
	err := c.Do(ctx, `query Project($id: String!) { project(id: $id) { `+projectFields+` } }`, map[string]any{"id": id}, &result)
	if err != nil {
		return nil, err
	}
	if result.Project == nil {
		return nil, ErrNotFound
	}
	return result.Project, nil
}

func (c *Client) ProjectBySlugID(ctx context.Context, slugID string) (*Project, error) {
	var result struct {
		Projects struct {
			Nodes []Project `json:"nodes"`
		} `json:"projects"`
	}
	variables := map[string]any{"filter": map[string]any{"slugId": map[string]any{"eq": slugID}}}
	err := c.Do(ctx, `query ProjectBySlugID($filter: ProjectFilter!) { projects(first: 2, filter: $filter) { nodes { `+projectFields+` } } }`, variables, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Projects.Nodes) == 0 {
		return nil, ErrNotFound
	}
	if len(result.Projects.Nodes) > 1 {
		return nil, fmt.Errorf("more than one Linear project has slug ID %q", slugID)
	}
	return &result.Projects.Nodes[0], nil
}

type IssueLabel struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	Description *string `json:"description"`
	Team        *struct {
		ID string `json:"id"`
	} `json:"team"`
}

const issueLabelFields = `id name color description team { id }`

func (c *Client) IssueLabel(ctx context.Context, id string) (*IssueLabel, error) {
	var result struct {
		Label *IssueLabel `json:"issueLabel"`
	}
	err := c.Do(ctx, `query IssueLabel($id: String!) { issueLabel(id: $id) { `+issueLabelFields+` } }`, map[string]any{"id": id}, &result)
	if err != nil {
		return nil, err
	}
	if result.Label == nil {
		return nil, ErrNotFound
	}
	return result.Label, nil
}

func (c *Client) CreateIssueLabel(ctx context.Context, input map[string]any) (*IssueLabel, error) {
	var result struct {
		Create struct {
			Success bool        `json:"success"`
			Label   *IssueLabel `json:"issueLabel"`
		} `json:"issueLabelCreate"`
	}
	err := c.Do(ctx, `mutation IssueLabelCreate($input: IssueLabelCreateInput!) { issueLabelCreate(input: $input) { success issueLabel { `+issueLabelFields+` } } }`, map[string]any{"input": input}, &result)
	if err != nil {
		return nil, err
	}
	if !result.Create.Success || result.Create.Label == nil {
		return nil, errors.New("Linear did not create the issue label")
	}
	return result.Create.Label, nil
}

func (c *Client) UpdateIssueLabel(ctx context.Context, id string, input map[string]any) (*IssueLabel, error) {
	var result struct {
		Update struct {
			Success bool        `json:"success"`
			Label   *IssueLabel `json:"issueLabel"`
		} `json:"issueLabelUpdate"`
	}
	err := c.Do(ctx, `mutation IssueLabelUpdate($id: String!, $input: IssueLabelUpdateInput!) { issueLabelUpdate(id: $id, input: $input) { success issueLabel { `+issueLabelFields+` } } }`, map[string]any{"id": id, "input": input}, &result)
	if err != nil {
		return nil, err
	}
	if !result.Update.Success || result.Update.Label == nil {
		return nil, errors.New("Linear did not update the issue label")
	}
	return result.Update.Label, nil
}

func (c *Client) DeleteIssueLabel(ctx context.Context, id string) error {
	var result struct {
		Delete struct {
			Success bool `json:"success"`
		} `json:"issueLabelDelete"`
	}
	err := c.Do(ctx, `mutation IssueLabelDelete($id: String!) { issueLabelDelete(id: $id) { success } }`, map[string]any{"id": id}, &result)
	if err != nil {
		return err
	}
	if !result.Delete.Success {
		return errors.New("Linear did not delete the issue label")
	}
	return nil
}

type CustomView struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Color       *string         `json:"color"`
	Icon        *string         `json:"icon"`
	Shared      bool            `json:"shared"`
	SlugID      string          `json:"slugId"`
	FilterData  json.RawMessage `json:"filterData"`
	Team        *struct {
		ID string `json:"id"`
	} `json:"team"`
}

const customViewFields = `id name description color icon shared slugId filterData team { id }`

func (c *Client) CustomView(ctx context.Context, id string) (*CustomView, error) {
	var result struct {
		View *CustomView `json:"customView"`
	}
	err := c.Do(ctx, `query CustomView($id: String!) { customView(id: $id) { `+customViewFields+` } }`, map[string]any{"id": id}, &result)
	if err != nil {
		return nil, err
	}
	if result.View == nil {
		return nil, ErrNotFound
	}
	return result.View, nil
}

func (c *Client) CreateCustomView(ctx context.Context, input map[string]any) (*CustomView, error) {
	var result struct {
		Create struct {
			Success bool        `json:"success"`
			View    *CustomView `json:"customView"`
		} `json:"customViewCreate"`
	}
	err := c.Do(ctx, `mutation CustomViewCreate($input: CustomViewCreateInput!) { customViewCreate(input: $input) { success customView { `+customViewFields+` } } }`, map[string]any{"input": input}, &result)
	if err != nil {
		return nil, err
	}
	if !result.Create.Success || result.Create.View == nil {
		return nil, errors.New("Linear did not create the custom view")
	}
	return result.Create.View, nil
}

func (c *Client) UpdateCustomView(ctx context.Context, id string, input map[string]any) (*CustomView, error) {
	var result struct {
		Update struct {
			Success bool        `json:"success"`
			View    *CustomView `json:"customView"`
		} `json:"customViewUpdate"`
	}
	err := c.Do(ctx, `mutation CustomViewUpdate($id: String!, $input: CustomViewUpdateInput!) { customViewUpdate(id: $id, input: $input) { success customView { `+customViewFields+` } } }`, map[string]any{"id": id, "input": input}, &result)
	if err != nil {
		return nil, err
	}
	if !result.Update.Success || result.Update.View == nil {
		return nil, errors.New("Linear did not update the custom view")
	}
	return result.Update.View, nil
}

func (c *Client) DeleteCustomView(ctx context.Context, id string) error {
	var result struct {
		Delete struct {
			Success bool `json:"success"`
		} `json:"customViewDelete"`
	}
	err := c.Do(ctx, `mutation CustomViewDelete($id: String!) { customViewDelete(id: $id) { success } }`, map[string]any{"id": id}, &result)
	if err != nil {
		return err
	}
	if !result.Delete.Success {
		return errors.New("Linear did not delete the custom view")
	}
	return nil
}
