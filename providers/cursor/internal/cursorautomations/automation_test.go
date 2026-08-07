package cursorautomations

import (
	"encoding/json"
	"strings"
	"testing"
)

func boolPtr(v bool) *bool { return &v }

// validConfig is a minimally complete configuration for tests to vary.
func validConfig() Config {
	return Config{
		Name:          "Daily review",
		Scope:         "TEAM_VISIBLE",
		Model:         "gpt-5.5-high",
		Enabled:       boolPtr(true),
		MemoryEnabled: boolPtr(true),
	}
}

func TestNewAutomationTrimsTrailingNewlines(t *testing.T) {
	automation, err := NewAutomation(validConfig(), "Review recent changes.\n\n")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}
	if automation.Prompt != "Review recent changes." {
		t.Errorf("prompt = %q, want the trailing newlines trimmed", automation.Prompt)
	}
}

func TestNewAutomationRejectsIncompleteConfig(t *testing.T) {
	tests := map[string]struct {
		mutate     func(*Config)
		prompt     string
		wantSubstr string
	}{
		"missing name":   {func(c *Config) { c.Name = "" }, "p", "name"},
		"missing scope":  {func(c *Config) { c.Scope = "" }, "p", "scope"},
		"missing model":  {func(c *Config) { c.Model = "" }, "p", "model"},
		"missing prompt": {func(*Config) {}, "   \n ", "prompt"},
		// An unset flag must not be silently read as false: an update replaces
		// the whole automation, so that would disable a live one.
		"unset enabled":       {func(c *Config) { c.Enabled = nil }, "p", "enabled"},
		"unset memoryEnabled": {func(c *Config) { c.MemoryEnabled = nil }, "p", "memoryEnabled"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			config := validConfig()
			tt.mutate(&config)

			_, err := NewAutomation(config, tt.prompt)
			if err == nil {
				t.Fatalf("expected an error naming %q", tt.wantSubstr)
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error = %q, want it to name %q", err, tt.wantSubstr)
			}
		})
	}
}

func TestNewAutomationValidatesTriggersAndActions(t *testing.T) {
	tests := map[string]struct {
		config  func(Config) Config
		wantErr bool
	}{
		"known trigger": {func(c Config) Config {
			c.Triggers = []map[string]any{{"cron": map[string]any{"cron": "0 9 * * 1-5"}}}
			return c
		}, false},
		"unknown trigger": {func(c Config) Config {
			c.Triggers = []map[string]any{{"carrierPigeon": map[string]any{}}}
			return c
		}, true},
		"known action": {func(c Config) Config {
			c.Actions = []map[string]any{{"requestReviewers": map[string]any{}}}
			return c
		}, false},
		"unknown action": {func(c Config) Config {
			c.Actions = []map[string]any{{"sendFax": map[string]any{}}}
			return c
		}, true},
		// A oneof carrying two keys is ambiguous, and the server would silently
		// pick one.
		"trigger with two keys": {func(c Config) Config {
			c.Triggers = []map[string]any{{"cron": map[string]any{}, "git": map[string]any{}}}
			return c
		}, true},
		"trigger with no keys": {func(c Config) Config {
			c.Triggers = []map[string]any{{}}
			return c
		}, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewAutomation(tt.config(validConfig()), "prompt")
			if tt.wantErr && err == nil {
				t.Error("expected an error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestQualifiedScope(t *testing.T) {
	tests := map[string]string{
		"TEAM_VISIBLE":                  "AUTOMATION_SCOPE_TEAM_VISIBLE",
		"AUTOMATION_SCOPE_TEAM_VISIBLE": "AUTOMATION_SCOPE_TEAM_VISIBLE",
		"PRIVATE":                       "AUTOMATION_SCOPE_PRIVATE",
		"":                              "",
	}
	for input, want := range tests {
		if got := qualifiedScope(input); got != want {
			t.Errorf("qualifiedScope(%q) = %q, want %q", input, got, want)
		}
	}
}

// The create payload is the contract with Cursor, so pin its exact shape.
func TestCreatePayloadShape(t *testing.T) {
	config := validConfig()
	config.MemoryEnabled = boolPtr(false)
	config.Triggers = []map[string]any{{"cron": map[string]any{"cron": "0 9 * * 1-5"}}}
	config.Actions = []map[string]any{{"requestReviewers": map[string]any{}}}
	config.SlackNotifiedChannels = []string{"C123"}
	config.GitConfig = &GitConfig{Repos: []string{"https://github.com/benfdking/sdsf"}, Branch: "main"}

	automation, err := NewAutomation(config, "Review recent changes.")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}

	encoded, err := json.Marshal(automation.RenderCreatePayload(123))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["name"] != "Daily review" {
		t.Errorf("name = %v", got["name"])
	}
	if got["scope"] != "AUTOMATION_SCOPE_TEAM_VISIBLE" {
		t.Errorf("scope = %v, want the qualified enum", got["scope"])
	}
	if got["enabled"] != true {
		t.Errorf("enabled = %v", got["enabled"])
	}
	if got["teamId"] != float64(123) {
		t.Errorf("teamId = %v", got["teamId"])
	}
	if got["creationSource"] != creationSource {
		t.Errorf("creationSource = %v", got["creationSource"])
	}

	workflow, ok := got["workflow"].(map[string]any)
	if !ok {
		t.Fatalf("workflow missing from payload: %s", encoded)
	}
	// memoryEnabled must survive as an explicit false rather than being
	// omitted, or turning memory off would never take effect.
	if memoryEnabled, present := workflow["memoryEnabled"]; !present || memoryEnabled != false {
		t.Errorf("memoryEnabled = %v (present %v), want an explicit false", memoryEnabled, present)
	}
	if workflow["model"] != "gpt-5.5-high" {
		t.Errorf("model = %v", workflow["model"])
	}

	prompt := workflow["prompts"].([]any)[0].(map[string]any)["prompt"].(string)
	if !strings.HasPrefix(prompt, managedPromptHeader) {
		t.Errorf("prompt is missing the managed header: %q", prompt)
	}
	if !strings.HasSuffix(prompt, "Review recent changes.") {
		t.Errorf("prompt body was not preserved: %q", prompt)
	}

	gitConfig := workflow["gitConfig"].(map[string]any)
	if gitConfig["branch"] != "main" {
		t.Errorf("gitConfig.branch = %v", gitConfig["branch"])
	}
}

// Empty collections are omitted so the server clears them, which is what a
// caller that set none means.
func TestCreatePayloadOmitsEmptyCollections(t *testing.T) {
	automation, err := NewAutomation(validConfig(), "prompt")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}

	encoded, err := json.Marshal(automation.RenderCreatePayload(1))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got struct {
		Workflow map[string]any `json:"workflow"`
	}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"triggers", "actions", "slackNotifiedChannels", "gitConfig"} {
		if _, present := got.Workflow[key]; present {
			t.Errorf("%q should be omitted when empty: %s", key, encoded)
		}
	}
}

func TestUpdateRequestCarriesAutomationID(t *testing.T) {
	automation, err := NewAutomation(validConfig(), "prompt")
	if err != nil {
		t.Fatalf("NewAutomation: %v", err)
	}

	request := automation.toUpdateRequest("auto-1")
	if request.AutomationID != "auto-1" {
		t.Errorf("automationId = %q", request.AutomationID)
	}
	if request.Scope != "AUTOMATION_SCOPE_TEAM_VISIBLE" {
		t.Errorf("scope = %q, want the qualified enum", request.Scope)
	}
	if !request.Enabled {
		t.Error("enabled = false, want true")
	}
}
