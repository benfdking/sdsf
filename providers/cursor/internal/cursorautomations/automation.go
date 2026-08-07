// Package cursorautomations models Cursor Automations and talks to the
// endpoints the Cursor dashboard uses to manage them.
//
// Cursor publishes no automations management API. Everything here is written
// against the request and response shapes the dashboard exchanges with
// cursor.com/api/automations, so it can change without notice.
package cursorautomations

import (
	"fmt"
	"strings"
)

// Automation is a single automation's desired state.
type Automation struct {
	Config Config
	Prompt string
}

// Config is everything about an automation except its prompt body.
type Config struct {
	Name  string
	Scope string
	Model string

	// Triggers and Actions are the API's oneof maps: exactly one key each,
	// naming the integration, with that integration's settings as the value.
	// They stay untyped so a new trigger or action type works without a change
	// here — validate only checks the key is one this package recognises.
	Triggers []map[string]any
	Actions  []map[string]any

	// Enabled and MemoryEnabled are pointers because an unset value has to be
	// distinguishable from an explicit false. Updates replace the whole
	// automation, so treating "unset" as false would let a caller that simply
	// forgot the field silently switch off a live automation.
	Enabled       *bool
	MemoryEnabled *bool

	SlackNotifiedChannels []string
	GitConfig             *GitConfig
}

// GitConfig limits an automation to a set of repositories and a base branch.
type GitConfig struct {
	Repos  []string `json:"repos,omitempty"`
	Branch string   `json:"branch,omitempty"`
}

// NewAutomation builds a validated automation. Trailing newlines are trimmed
// from the prompt so a file that ends in one produces the same payload as a
// string literal that does not.
func NewAutomation(config Config, prompt string) (Automation, error) {
	automation := Automation{
		Config: config,
		Prompt: strings.TrimRight(prompt, "\n"),
	}
	if err := automation.validate(); err != nil {
		return Automation{}, err
	}
	return automation, nil
}

// isEnabled and isMemoryEnabled read the optional flags, treating unset as
// false. validate rejects unset, so these only see a nil pointer on an
// Automation built without NewAutomation.
func (c Config) isEnabled() bool       { return c.Enabled != nil && *c.Enabled }
func (c Config) isMemoryEnabled() bool { return c.MemoryEnabled != nil && *c.MemoryEnabled }

// supportedTriggers and supportedActions are the oneof keys observed on the
// automations API. Validating against them turns a typo into a local error
// rather than a payload the server accepts and then ignores.
var (
	supportedTriggers = map[string]bool{
		"cron":               true,
		"git":                true,
		"linear":             true,
		"slackReactionAdded": true,
		"slackTrigger":       true,
	}

	supportedActions = map[string]bool{
		"mcp":              true,
		"prComment":        true,
		"readSlack":        true,
		"requestReviewers": true,
		"slack":            true,
	}
)

// scopePrefix is the enum prefix the API expects on a scope value.
const scopePrefix = "AUTOMATION_SCOPE_"

// qualifiedScope accepts either the bare scope ("TEAM_VISIBLE") or the full
// enum ("AUTOMATION_SCOPE_TEAM_VISIBLE") and returns the full enum.
func qualifiedScope(scope string) string {
	if scope == "" || strings.HasPrefix(scope, scopePrefix) {
		return scope
	}
	return scopePrefix + scope
}

// validate reports the first reason an automation could not be synced.
func (a Automation) validate() error {
	required := []struct {
		field   string
		missing bool
		hint    string
	}{
		{"name", a.Config.Name == "", ""},
		{"scope", a.Config.Scope == "", ""},
		{"model", a.Config.Model == "", ""},
		{"enabled", a.Config.Enabled == nil, " (set true or false)"},
		{"memoryEnabled", a.Config.MemoryEnabled == nil, " (set true or false)"},
		{"prompt", strings.TrimSpace(a.Prompt) == "", ""},
	}
	for _, r := range required {
		if r.missing {
			return fmt.Errorf("missing required field: %s%s", r.field, r.hint)
		}
	}

	for i, trigger := range a.Config.Triggers {
		if err := validateOneOf(trigger, supportedTriggers); err != nil {
			return fmt.Errorf("trigger %d: %w", i, err)
		}
	}
	for i, action := range a.Config.Actions {
		if err := validateOneOf(action, supportedActions); err != nil {
			return fmt.Errorf("action %d: %w", i, err)
		}
	}
	return nil
}

// validateOneOf checks a oneof map carries exactly one recognised key.
func validateOneOf(entry map[string]any, supported map[string]bool) error {
	if len(entry) != 1 {
		return fmt.Errorf("expected exactly one type key, got %d", len(entry))
	}
	for key := range entry {
		if !supported[key] {
			return fmt.Errorf("unknown type %q", key)
		}
	}
	return nil
}
