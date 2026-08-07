package provider

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/benfdking/sdsf/providers/linear/internal/linear"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDecodeFilterJSON(t *testing.T) {
	filter, err := decodeFilterJSON(types.StringValue(`{"priority":{"lte":2},"completedAt":{"null":true}}`))
	if err != nil {
		t.Fatalf("decodeFilterJSON returned error: %v", err)
	}
	want := map[string]any{
		"priority":    map[string]any{"lte": float64(2)},
		"completedAt": map[string]any{"null": true},
	}
	if !reflect.DeepEqual(filter, want) {
		t.Fatalf("filter = %#v, want %#v", filter, want)
	}
}

func TestDecodeFilterJSONRejectsNonObject(t *testing.T) {
	for _, value := range []string{`null`, `[1,2]`, `not-json`} {
		if _, err := decodeFilterJSON(types.StringValue(value)); err == nil {
			t.Errorf("decodeFilterJSON(%q) unexpectedly succeeded", value)
		}
	}
}

func TestFilterStatePreservesEquivalentPriorJSON(t *testing.T) {
	prior := types.StringValue("{\n  \"priority\": { \"lte\": 2 }\n}")
	got := filterState(json.RawMessage(`{"priority":{"lte":2}}`), prior)
	if !got.Equal(prior) {
		t.Fatalf("filterState = %q, want prior value %q", got.ValueString(), prior.ValueString())
	}
}

func TestCustomViewStateHandlesWorkspaceScope(t *testing.T) {
	view := &linear.CustomView{ID: "view-id", Name: "Security", Shared: true, SlugID: "security", FilterData: json.RawMessage(`{}`)}
	state := customViewState(view, customViewModel{})
	if !state.TeamID.IsNull() || state.ID.ValueString() != "view-id" || !state.Shared.ValueBool() {
		t.Fatalf("unexpected state: %#v", state)
	}
}

func TestLabelColorValidation(t *testing.T) {
	for _, color := range []string{"#5E6AD2", "#ffffff"} {
		if !validLabelPlan(issueLabelModel{Color: types.StringValue(color)}) {
			t.Errorf("expected %q to be valid", color)
		}
	}
	for _, color := range []string{"5E6AD2", "#fff", "#GGGGGG"} {
		if validLabelPlan(issueLabelModel{Color: types.StringValue(color)}) {
			t.Errorf("expected %q to be invalid", color)
		}
	}
}

func TestProjectStateHandlesNullableFieldsAndTeams(t *testing.T) {
	project := &linear.Project{ID: "project-id", Name: "Launch", SlugID: "launch-abc123", URL: "https://linear.app/project/launch-abc123", Progress: 0.5}
	project.Teams.Nodes = append(project.Teams.Nodes, struct {
		ID string `json:"id"`
	}{ID: "team-id"})

	state, err := projectState(t.Context(), project)
	if err != nil {
		t.Fatalf("projectState returned error: %v", err)
	}
	if state.ID.ValueString() != "project-id" || !state.Description.IsNull() || !state.StatusID.IsNull() || !state.LeadID.IsNull() {
		t.Fatalf("unexpected project state: %#v", state)
	}
	var teamIDs []string
	diagnostics := state.TeamIDs.ElementsAs(t.Context(), &teamIDs, false)
	if diagnostics.HasError() || !reflect.DeepEqual(teamIDs, []string{"team-id"}) {
		t.Fatalf("team IDs = %#v, diagnostics = %v", teamIDs, diagnostics)
	}
}
