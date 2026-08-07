package cursorautomations

import (
	"encoding/json"
	"testing"
)

func TestParseListExtractsNameAndID(t *testing.T) {
	raw := []byte(`{"workflows":[
		{"workflow":{"name":"Daily review","automationId":"auto-1","model":"gpt-5.5-high"}},
		{"workflow":{"name":"Nightly sweep","automationId":"auto-2"}}
	]}`)

	automations, err := ParseList(raw)
	if err != nil {
		t.Fatalf("ParseList: %v", err)
	}
	if len(automations) != 2 {
		t.Fatalf("got %d automations, want 2", len(automations))
	}
	if automations[0].Name != "Daily review" || automations[0].AutomationID != "auto-1" {
		t.Errorf("first automation = %+v", automations[0])
	}

	// Raw keeps the whole object, including fields this package does not model.
	var raw0 map[string]any
	if err := json.Unmarshal(automations[0].Raw, &raw0); err != nil {
		t.Fatalf("unmarshal Raw: %v", err)
	}
	if raw0["model"] != "gpt-5.5-high" {
		t.Errorf("Raw dropped unmodelled fields: %s", automations[0].Raw)
	}
}

func TestParseListReturnsEmptySliceForEmptyList(t *testing.T) {
	automations, err := ParseList([]byte(`{"workflows":[]}`))
	if err != nil {
		t.Fatalf("ParseList: %v", err)
	}
	if len(automations) != 0 {
		t.Errorf("got %d automations, want 0", len(automations))
	}
}

// Failing closed matters: reading a reshaped response as "no automations" would
// make a caller re-create automations that already exist.
func TestParseListFailsClosedOnMissingWorkflows(t *testing.T) {
	for name, raw := range map[string]string{
		"empty object":   `{}`,
		"renamed key":    `{"automations":[]}`,
		"null workflows": `{"workflows":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseList([]byte(raw)); err == nil {
				t.Errorf("expected an error for %s", raw)
			}
		})
	}
}

func TestParseListRejectsMalformedJSON(t *testing.T) {
	if _, err := ParseList([]byte(`not json`)); err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

func TestParseListToleratesEntryWithoutWorkflow(t *testing.T) {
	automations, err := ParseList([]byte(`{"workflows":[{}]}`))
	if err != nil {
		t.Fatalf("ParseList: %v", err)
	}
	if len(automations) != 1 || automations[0].Name != "" {
		t.Errorf("automations = %+v, want one empty entry", automations)
	}
}
