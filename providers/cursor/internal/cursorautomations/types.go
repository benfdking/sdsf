package cursorautomations

import (
	"fmt"
	"strings"
)

// Automation is one automation defined in code: the parsed automation.yaml plus
// the prompt body from the sibling prompt.md.
type Automation struct {
	Dir    string // directory name, e.g. "test-automation"
	Config Config
	Prompt string
}

// Config is the YAML in an automation.yaml file. Field names mirror the Cursor
// API so the mapping to a request payload stays obvious.
type Config struct {
	Name     string           `yaml:"name"`
	Scope    string           `yaml:"scope"`
	Model    string           `yaml:"model"`
	Triggers []map[string]any `yaml:"triggers"`
	Actions  []map[string]any `yaml:"actions"`

	// Enabled and MemoryEnabled are required pointers: an omitted value is distinct
	// from explicit false. Update replaces the whole workflow, so a plain bool would
	// let an omitted field silently disable a live automation.
	Enabled       *bool `yaml:"enabled"`
	MemoryEnabled *bool `yaml:"memoryEnabled"`

	// Carried through verbatim and modelled so a full sync doesn't drop it.
	SlackNotifiedChannels []string   `yaml:"slackNotifiedChannels"`
	GitConfig             *GitConfig `yaml:"gitConfig"`
}

// GitConfig selects the repositories and base branch available to an automation.
type GitConfig struct {
	Repos  []string `json:"repos,omitempty" yaml:"repos"`
	Branch string   `json:"branch,omitempty" yaml:"branch"`
}

// NewAutomation validates and constructs an automation without requiring the
// legacy automation.yaml/prompt.md directory representation.
func NewAutomation(config Config, prompt string) (Automation, error) {
	automation := Automation{
		Dir:    config.Name,
		Config: config,
		Prompt: strings.TrimRight(prompt, "\n"),
	}
	if err := automation.validate(); err != nil {
		return Automation{}, err
	}
	return automation, nil
}

// enabled / memoryEnabled dereference their required pointers; nil counts as false.
func (c Config) enabled() bool       { return c.Enabled != nil && *c.Enabled }
func (c Config) memoryEnabled() bool { return c.MemoryEnabled != nil && *c.MemoryEnabled }

// knownTriggers / knownActions are the oneof keys the Cursor API accepts, as
// observed from list-automations. Validation rejects unknown keys so a typo
// fails locally rather than silently producing a no-op payload.
var knownTriggers = map[string]bool{
	"slackReactionAdded": true,
	"slackTrigger":       true,
	"cron":               true,
	"git":                true,
	"linear":             true,
}

var knownActions = map[string]bool{
	"slack":            true,
	"readSlack":        true,
	"mcp":              true,
	"prComment":        true,
	"requestReviewers": true,
}

// normaliseScope accepts either the short form ("TEAM_VISIBLE") or the full
// enum ("AUTOMATION_SCOPE_TEAM_VISIBLE") and returns the full enum the API wants.
func normaliseScope(scope string) string {
	if scope == "" {
		return ""
	}
	if strings.HasPrefix(scope, "AUTOMATION_SCOPE_") {
		return scope
	}
	return "AUTOMATION_SCOPE_" + scope
}

// updateRequest is the body POSTed to /update-automation. enabled is always sent
// so the YAML stays authoritative for whether the automation is active.
type updateRequest struct {
	Name         string        `json:"name"`
	Enabled      bool          `json:"enabled"`
	Workflow     innerWorkflow `json:"workflow"`
	AutomationID string        `json:"automationId"`
	Scope        string        `json:"scope"`
}

// innerWorkflow is the nested "workflow" object holding the actual config.
// Update replaces the whole workflow: the dashboard sends the full desired state
// each save, omitting empty arrays (which clears them) but always sending
// memoryEnabled. So slices keep omitempty (omitting an empty one clears it), and
// memoryEnabled drops omitempty so disabling it actually takes effect rather than
// relying on the backend's default for an omitted bool.
type innerWorkflow struct {
	Prompts               []promptStep     `json:"prompts,omitempty"`
	Model                 string           `json:"model,omitempty"`
	Triggers              []map[string]any `json:"triggers,omitempty"`
	Actions               []map[string]any `json:"actions,omitempty"`
	MemoryEnabled         bool             `json:"memoryEnabled"`
	SlackNotifiedChannels []string         `json:"slackNotifiedChannels,omitempty"`
	GitConfig             *GitConfig       `json:"gitConfig,omitempty"`
}

type promptStep struct {
	Prompt string `json:"prompt"`
}

// RenderCreatePayload returns the body that would be sent to /create-automation,
// for dry-run inspection. It's representative for both create and update: which
// one runs is decided at apply time by matching the name against the live list.
func (a Automation) RenderCreatePayload(teamID int64) any {
	return a.toCreateRequest(teamID)
}

const managedPromptHeader = "<!-- Managed by internal-ai/cursor-automations. Don't edit here; changes are overwritten on the next sync. -->"

func (a Automation) promptWithHeader() string {
	return managedPromptHeader + "\n\n" + a.Prompt
}

// workflowPayload builds the nested workflow object shared by create and update.
func (a Automation) workflowPayload() innerWorkflow {
	return innerWorkflow{
		Prompts:               []promptStep{{Prompt: a.promptWithHeader()}},
		Model:                 a.Config.Model,
		Triggers:              a.Config.Triggers,
		Actions:               a.Config.Actions,
		MemoryEnabled:         a.Config.memoryEnabled(),
		SlackNotifiedChannels: a.Config.SlackNotifiedChannels,
		GitConfig:             a.Config.GitConfig,
	}
}

func (a Automation) toUpdateRequest(automationID string) updateRequest {
	return updateRequest{
		Name:         a.Config.Name,
		Enabled:      a.Config.enabled(),
		AutomationID: automationID,
		Scope:        normaliseScope(a.Config.Scope),
		Workflow:     a.workflowPayload(),
	}
}

// creationSource tags how an automation was created. The portal-web value is
// what the dashboard sends; the API accepts it.
const creationSource = "AUTOMATION_CREATION_SOURCE_PORTAL_WEB"

// createRequest is the body POSTed to /create-automation. It carries no
// automationId (the server assigns one) but adds teamId and creationSource.
type createRequest struct {
	Name           string        `json:"name"`
	Enabled        bool          `json:"enabled"`
	Workflow       innerWorkflow `json:"workflow"`
	Scope          string        `json:"scope"`
	TeamID         int64         `json:"teamId"`
	CreationSource string        `json:"creationSource"`
}

func (a Automation) toCreateRequest(teamID int64) createRequest {
	return createRequest{
		Name:           a.Config.Name,
		Enabled:        a.Config.enabled(),
		Scope:          normaliseScope(a.Config.Scope),
		TeamID:         teamID,
		CreationSource: creationSource,
		Workflow:       a.workflowPayload(),
	}
}

// validate checks an automation is well-formed before we try to sync it.
func (a Automation) validate() error {
	if a.Config.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if a.Config.Enabled == nil {
		return fmt.Errorf("missing required field: enabled (set true or false)")
	}
	if a.Config.MemoryEnabled == nil {
		return fmt.Errorf("missing required field: memoryEnabled (set true or false)")
	}
	if a.Config.Scope == "" {
		return fmt.Errorf("missing required field: scope")
	}
	if a.Config.Model == "" {
		return fmt.Errorf("missing required field: model")
	}
	if strings.TrimSpace(a.Prompt) == "" {
		return fmt.Errorf("prompt.md is empty")
	}
	for i, t := range a.Config.Triggers {
		if err := validateOneOf(t, knownTriggers); err != nil {
			return fmt.Errorf("trigger %d: %w", i, err)
		}
	}
	for i, act := range a.Config.Actions {
		if err := validateOneOf(act, knownActions); err != nil {
			return fmt.Errorf("action %d: %w", i, err)
		}
	}
	return nil
}

func validateOneOf(entry map[string]any, known map[string]bool) error {
	if len(entry) != 1 {
		return fmt.Errorf("expected exactly one type key, got %d", len(entry))
	}
	for k := range entry {
		if !known[k] {
			return fmt.Errorf("unknown type %q", k)
		}
	}
	return nil
}
