package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/benfdking/sdsf/providers/linear/internal/linear"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &customViewResource{}
var _ resource.ResourceWithConfigure = &customViewResource{}
var _ resource.ResourceWithImportState = &customViewResource{}

type customViewResource struct {
	client *linear.Client
}

type customViewModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Color       types.String `tfsdk:"color"`
	Icon        types.String `tfsdk:"icon"`
	Shared      types.Bool   `tfsdk:"shared"`
	TeamID      types.String `tfsdk:"team_id"`
	FilterJSON  types.String `tfsdk:"filter_json"`
	SlugID      types.String `tfsdk:"slug_id"`
}

func NewCustomViewResource() resource.Resource { return &customViewResource{} }

func (r *customViewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_view"
}

func (r *customViewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Linear custom issue view.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "The custom view's UUID."},
			"name":        schema.StringAttribute{Required: true, Description: "Custom view name."},
			"description": schema.StringAttribute{Optional: true, Description: "Custom view description."},
			"color":       schema.StringAttribute{Optional: true, Description: "Color of the custom view icon."},
			"icon":        schema.StringAttribute{Optional: true, Description: "Linear icon name for the custom view."},
			"shared":      schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the view is shared with the workspace."},
			"team_id":     schema.StringAttribute{Optional: true, Description: "Team UUID associated with the view. Omit for a workspace view."},
			"filter_json": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Linear IssueFilter encoded as a JSON object. This escape hatch tracks Linear's evolving GraphQL filter schema without provider upgrades.",
			},
			"slug_id": schema.StringAttribute{Computed: true, Description: "Linear's URL slug identifier for the view."},
		},
	}
}

func (r *customViewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*clientData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "The Linear provider returned an unexpected resource configuration type.")
		return
	}
	r.client = data.client
}

func decodeFilterJSON(value types.String) (map[string]any, error) {
	if value.IsNull() || value.ValueString() == "" {
		return map[string]any{}, nil
	}
	var filter map[string]any
	if err := json.Unmarshal([]byte(value.ValueString()), &filter); err != nil {
		return nil, fmt.Errorf("filter_json must contain valid JSON: %w", err)
	}
	if filter == nil {
		return nil, errors.New("filter_json must be a JSON object, not null")
	}
	return filter, nil
}

func customViewInput(plan customViewModel) (map[string]any, error) {
	filter, err := decodeFilterJSON(plan.FilterJSON)
	if err != nil {
		return nil, err
	}
	input := map[string]any{
		"name":       plan.Name.ValueString(),
		"shared":     plan.Shared.ValueBool(),
		"filterData": filter,
	}
	optionalStringInput(input, "description", plan.Description)
	optionalStringInput(input, "color", plan.Color)
	optionalStringInput(input, "icon", plan.Icon)
	optionalStringInput(input, "teamId", plan.TeamID)
	return input, nil
}

func optionalStringInput(input map[string]any, name string, value types.String) {
	if value.IsNull() {
		input[name] = nil
	} else {
		input[name] = value.ValueString()
	}
}

func stringPointerValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func filterState(raw json.RawMessage, prior types.String) types.String {
	var remote any
	if len(raw) == 0 || string(raw) == "null" {
		remote = map[string]any{}
	} else if err := json.Unmarshal(raw, &remote); err != nil {
		return types.StringValue(string(raw))
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		var previous any
		if json.Unmarshal([]byte(prior.ValueString()), &previous) == nil && reflect.DeepEqual(previous, remote) {
			return prior
		}
	}
	canonical, err := json.Marshal(remote)
	if err != nil {
		return types.StringValue("{}")
	}
	return types.StringValue(string(canonical))
}

func customViewState(view *linear.CustomView, prior customViewModel) customViewModel {
	state := customViewModel{
		ID:          types.StringValue(view.ID),
		Name:        types.StringValue(view.Name),
		Description: stringPointerValue(view.Description),
		Color:       stringPointerValue(view.Color),
		Icon:        stringPointerValue(view.Icon),
		Shared:      types.BoolValue(view.Shared),
		FilterJSON:  filterState(view.FilterData, prior.FilterJSON),
		SlugID:      types.StringValue(view.SlugID),
	}
	if view.Team == nil {
		state.TeamID = types.StringNull()
	} else {
		state.TeamID = types.StringValue(view.Team.ID)
	}
	return state
}

func (r *customViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customViewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, err := customViewInput(plan)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("filter_json"), "Invalid Linear view filter", err.Error())
		return
	}
	view, err := r.client.CreateCustomView(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Linear custom view", err.Error())
		return
	}
	state := customViewState(view, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customViewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.CustomView(ctx, state.ID.ValueString())
	if errors.Is(err, linear.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Linear custom view", err.Error())
		return
	}
	newState := customViewState(view, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *customViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customViewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, err := customViewInput(plan)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("filter_json"), "Invalid Linear view filter", err.Error())
		return
	}
	view, err := r.client.UpdateCustomView(ctx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Linear custom view", err.Error())
		return
	}
	state := customViewState(view, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customViewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCustomView(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, linear.ErrNotFound) {
		resp.Diagnostics.AddError("Unable to delete Linear custom view", err.Error())
	}
}

func (r *customViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
