package cursor

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Run statuses. CREATING and RUNNING are in-flight; the rest are terminal.
const (
	RunStatusCreating  = "CREATING"
	RunStatusRunning   = "RUNNING"
	RunStatusFinished  = "FINISHED"
	RunStatusError     = "ERROR"
	RunStatusCancelled = "CANCELLED"
	RunStatusExpired   = "EXPIRED"
)

// IsTerminal reports whether a run status means no further work will happen.
// Unknown statuses are treated as non-terminal so a newly introduced in-flight
// state doesn't make a wait exit early and report success.
func IsTerminal(status string) bool {
	switch status {
	case RunStatusFinished, RunStatusError, RunStatusCancelled, RunStatusExpired:
		return true
	default:
		return false
	}
}

// Repo is a repository an agent works in.
type Repo struct {
	URL         string `json:"url"`
	StartingRef string `json:"startingRef,omitempty"`
	PRURL       string `json:"prUrl,omitempty"`
}

// Env selects the execution environment: cloud, pool, or machine. Name targets
// a specific pool or machine and is ignored for cloud.
type Env struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// Model names the model plus any parameter overrides.
type Model struct {
	ID     string       `json:"id"`
	Params []ModelParam `json:"params,omitempty"`
}

// ModelParam is a single model parameter override, e.g. effort=high.
type ModelParam struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// Prompt is the instruction given to an agent.
type Prompt struct {
	Text string `json:"text"`
}

// GitBranch is a branch an agent pushed, with its PR when one was opened.
type GitBranch struct {
	RepoURL string `json:"repoUrl"`
	Branch  string `json:"branch,omitempty"`
	PRURL   string `json:"prUrl,omitempty"`
}

// Git holds the branches produced by a run.
type Git struct {
	Branches []GitBranch `json:"branches,omitempty"`
}

// Agent is a long-lived container for runs. Its own status is only
// ACTIVE/ARCHIVED — the interesting state lives on the run.
type Agent struct {
	ID                  string `json:"id"`
	Name                string `json:"name,omitempty"`
	Status              string `json:"status,omitempty"`
	Env                 *Env   `json:"env,omitempty"`
	Repos               []Repo `json:"repos,omitempty"`
	WorkOnCurrentBranch bool   `json:"workOnCurrentBranch,omitempty"`
	AutoCreatePR        bool   `json:"autoCreatePR,omitempty"`
	URL                 string `json:"url,omitempty"`
	CreatedAt           string `json:"createdAt,omitempty"`
	UpdatedAt           string `json:"updatedAt,omitempty"`
	LatestRunID         string `json:"latestRunId,omitempty"`
}

// Run is one execution of a prompt against an agent. DurationMs, Result, and
// Git are only populated once the run is terminal.
type Run struct {
	ID         string `json:"id"`
	AgentID    string `json:"agentId"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt,omitempty"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Result     string `json:"result,omitempty"`
	Git        *Git   `json:"git,omitempty"`
}

// CreateAgentRequest is the body of POST /v1/agents. Fields left at their zero
// value are omitted so the server applies its own defaults rather than having
// the CLI hardcode them.
type CreateAgentRequest struct {
	Prompt              Prompt            `json:"prompt"`
	Model               *Model            `json:"model,omitempty"`
	Name                string            `json:"name,omitempty"`
	Env                 *Env              `json:"env,omitempty"`
	Repos               []Repo            `json:"repos,omitempty"`
	WorkOnCurrentBranch bool              `json:"workOnCurrentBranch,omitempty"`
	AutoCreatePR        bool              `json:"autoCreatePR,omitempty"`
	SkipReviewerRequest bool              `json:"skipReviewerRequest,omitempty"`
	EnvVars             map[string]string `json:"envVars,omitempty"`
	Mode                string            `json:"mode,omitempty"`
	AgentID             string            `json:"agentId,omitempty"`
}

// CreateAgentResponse pairs the new agent with the run its creation started.
type CreateAgentResponse struct {
	Agent Agent `json:"agent"`
	Run   Run   `json:"run"`
}

// CreateRunRequest is the body of POST /v1/agents/{id}/runs.
type CreateRunRequest struct {
	Prompt Prompt `json:"prompt"`
	Mode   string `json:"mode,omitempty"`
}

// ListAgentsOptions filters and paginates GET /v1/agents.
type ListAgentsOptions struct {
	Limit           int
	Cursor          string
	IncludeArchived *bool
}

// AgentPage is one page of agents. NextCursor is empty on the last page.
type AgentPage struct {
	Items      []Agent `json:"items"`
	NextCursor string  `json:"nextCursor,omitempty"`
}

// RunPage is one page of runs. NextCursor is empty on the last page.
type RunPage struct {
	Items      []Run  `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// CreateAgent creates an agent and starts its first run.
func (c *Client) CreateAgent(ctx context.Context, req CreateAgentRequest) (*CreateAgentResponse, error) {
	var resp CreateAgentResponse
	if err := c.do(ctx, "POST", "/v1/agents", req, &resp); err != nil {
		return nil, err
	}
	// The run id is what everything downstream waits on, so a response missing
	// it is a contract break worth surfacing here rather than as a confusing
	// 404 on the next call.
	if resp.Run.ID == "" {
		return nil, fmt.Errorf("agent %s created but response contained no run id", resp.Agent.ID)
	}
	if resp.Run.AgentID == "" {
		resp.Run.AgentID = resp.Agent.ID
	}
	return &resp, nil
}

// CreateRun queues a follow-up prompt on an existing agent.
func (c *Client) CreateRun(ctx context.Context, agentID string, req CreateRunRequest) (*Run, error) {
	var resp struct {
		Run Run `json:"run"`
	}
	if err := c.do(ctx, "POST", "/v1/agents/"+url.PathEscape(agentID)+"/runs", req, &resp); err != nil {
		return nil, err
	}
	if resp.Run.ID == "" {
		return nil, fmt.Errorf("follow-up on agent %s returned no run id", agentID)
	}
	if resp.Run.AgentID == "" {
		resp.Run.AgentID = agentID
	}
	return &resp.Run, nil
}

// GetAgent fetches a single agent.
func (c *Client) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	var agent Agent
	if err := c.do(ctx, "GET", "/v1/agents/"+url.PathEscape(agentID), nil, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetRun fetches a single run. This is the authoritative status source, and the
// fallback whenever streaming is unavailable.
func (c *Client) GetRun(ctx context.Context, agentID, runID string) (*Run, error) {
	var run Run
	path := "/v1/agents/" + url.PathEscape(agentID) + "/runs/" + url.PathEscape(runID)
	if err := c.do(ctx, "GET", path, nil, &run); err != nil {
		return nil, err
	}
	if run.AgentID == "" {
		run.AgentID = agentID
	}
	if run.ID == "" {
		run.ID = runID
	}
	return &run, nil
}

// ListAgents returns one page of agents.
func (c *Client) ListAgents(ctx context.Context, opts ListAgentsOptions) (*AgentPage, error) {
	query := url.Values{}
	if opts.Limit > 0 {
		query.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Cursor != "" {
		query.Set("cursor", opts.Cursor)
	}
	if opts.IncludeArchived != nil {
		query.Set("includeArchived", strconv.FormatBool(*opts.IncludeArchived))
	}

	path := "/v1/agents"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var page AgentPage
	if err := c.do(ctx, "GET", path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// ListRuns returns one page of an agent's runs, newest first.
func (c *Client) ListRuns(ctx context.Context, agentID string, limit int, cursor string) (*RunPage, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		query.Set("cursor", cursor)
	}

	path := "/v1/agents/" + url.PathEscape(agentID) + "/runs"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var page RunPage
	if err := c.do(ctx, "GET", path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// CancelRun stops an in-flight run.
func (c *Client) CancelRun(ctx context.Context, agentID, runID string) error {
	path := "/v1/agents/" + url.PathEscape(agentID) + "/runs/" + url.PathEscape(runID) + "/cancel"
	return c.do(ctx, "POST", path, struct{}{}, nil)
}

// ResolveRunID returns runID unchanged, or looks up the agent's latest run when
// it is empty. This is what lets every wait/show command take a bare agent id.
func (c *Client) ResolveRunID(ctx context.Context, agentID, runID string) (string, error) {
	if runID != "" {
		return runID, nil
	}
	agent, err := c.GetAgent(ctx, agentID)
	if err != nil {
		return "", err
	}
	if agent.LatestRunID != "" {
		return agent.LatestRunID, nil
	}
	// Older agents may predate latestRunId; fall back to the runs listing.
	page, err := c.ListRuns(ctx, agentID, 1, "")
	if err != nil {
		return "", err
	}
	if len(page.Items) == 0 {
		return "", fmt.Errorf("agent %s has no runs", agentID)
	}
	return page.Items[0].ID, nil
}
