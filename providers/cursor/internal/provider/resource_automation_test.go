package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildAutomationFromTerraformValues(t *testing.T) {
	cronType := types.ObjectType{AttrTypes: map[string]attr.Type{"cron": types.StringType}}
	cron := types.ObjectValueMust(cronType.AttrTypes, map[string]attr.Value{
		"cron": types.StringValue("0 9 * * 1-5"),
	})
	trigger := types.ObjectValueMust(map[string]attr.Type{"cron": cronType}, map[string]attr.Value{
		"cron": cron,
	})
	emptyObject := types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{})
	action := types.ObjectValueMust(
		map[string]attr.Type{"requestReviewers": types.ObjectType{AttrTypes: map[string]attr.Type{}}},
		map[string]attr.Value{"requestReviewers": emptyObject},
	)

	gitConfigType := map[string]attr.Type{
		"repositories": types.SetType{ElemType: types.StringType},
		"branch":       types.StringType,
	}
	plan := automationModel{
		Name:          types.StringValue("Daily review"),
		Scope:         types.StringValue("TEAM_VISIBLE"),
		Model:         types.StringValue("gpt-5.5-high"),
		Prompt:        types.StringValue("Review changes."),
		Enabled:       types.BoolValue(true),
		MemoryEnabled: types.BoolValue(false),
		Triggers: types.DynamicValue(types.TupleValueMust(
			[]attr.Type{trigger.Type(context.Background())},
			[]attr.Value{trigger},
		)),
		Actions: types.DynamicValue(types.TupleValueMust(
			[]attr.Type{action.Type(context.Background())},
			[]attr.Value{action},
		)),
		SlackNotifiedChannels: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("C123")}),
		GitConfig: types.ObjectValueMust(gitConfigType, map[string]attr.Value{
			"repositories": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("https://github.com/benfdking/sdsf")}),
			"branch":       types.StringValue("main"),
		}),
	}

	automation, diagnostics := buildAutomation(context.Background(), plan)
	if diagnostics.HasError() {
		t.Fatalf("buildAutomation returned diagnostics: %v", diagnostics)
	}

	payload, err := json.Marshal(automation.RenderCreatePayload(123))
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	workflow := got["workflow"].(map[string]any)
	if got["name"] != "Daily review" || got["scope"] != "AUTOMATION_SCOPE_TEAM_VISIBLE" {
		t.Fatalf("unexpected top-level payload: %s", payload)
	}
	if workflow["memoryEnabled"] != false {
		t.Fatalf("memoryEnabled was not preserved: %s", payload)
	}
	if workflow["triggers"].([]any)[0].(map[string]any)["cron"].(map[string]any)["cron"] != "0 9 * * 1-5" {
		t.Fatalf("cron trigger was not preserved: %s", payload)
	}
	if workflow["gitConfig"].(map[string]any)["branch"] != "main" {
		t.Fatalf("git config was not preserved: %s", payload)
	}
}

func TestDynamicObjectListRejectsNonCollection(t *testing.T) {
	_, err := dynamicObjectList(types.DynamicValue(types.StringValue("not-a-list")))
	if err == nil {
		t.Fatal("expected a non-list dynamic value to fail")
	}
}
