package provider

import (
	"context"
	"errors"
	"regexp"

	"github.com/benfdking/sdsf/providers/linear/internal/linear"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &issueLabelResource{}
var _ resource.ResourceWithConfigure = &issueLabelResource{}
var _ resource.ResourceWithImportState = &issueLabelResource{}

type issueLabelResource struct {
	client *linear.Client
}

type issueLabelModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Color       types.String `tfsdk:"color"`
	Description types.String `tfsdk:"description"`
	TeamID      types.String `tfsdk:"team_id"`
}

func NewIssueLabelResource() resource.Resource { return &issueLabelResource{} }

func (r *issueLabelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_issue_label"
}

func (r *issueLabelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a team-scoped or workspace-scoped Linear issue label.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true, Description: "The label's UUID."},
			"name": schema.StringAttribute{Required: true, Description: "Label name."},
			"color": schema.StringAttribute{
				Required:    true,
				Description: "Label color as a six-digit hexadecimal value, for example #5E6AD2.",
			},
			"description": schema.StringAttribute{Optional: true, Description: "Label description."},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Description:   "Team UUID. Omit to create a workspace-scoped label.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *issueLabelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func labelCreateInput(plan issueLabelModel) map[string]any {
	input := map[string]any{"name": plan.Name.ValueString(), "color": plan.Color.ValueString()}
	if !plan.Description.IsNull() {
		input["description"] = plan.Description.ValueString()
	}
	if !plan.TeamID.IsNull() {
		input["teamId"] = plan.TeamID.ValueString()
	}
	return input
}

var hexColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func validLabelPlan(plan issueLabelModel) bool {
	return hexColorPattern.MatchString(plan.Color.ValueString())
}

func labelState(label *linear.IssueLabel) issueLabelModel {
	state := issueLabelModel{ID: types.StringValue(label.ID), Name: types.StringValue(label.Name), Color: types.StringValue(label.Color)}
	if label.Description == nil {
		state.Description = types.StringNull()
	} else {
		state.Description = types.StringValue(*label.Description)
	}
	if label.Team == nil {
		state.TeamID = types.StringNull()
	} else {
		state.TeamID = types.StringValue(label.Team.ID)
	}
	return state
}

func (r *issueLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan issueLabelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !validLabelPlan(plan) {
		resp.Diagnostics.AddAttributeError(path.Root("color"), "Invalid label color", "Color must be a six-digit hexadecimal value beginning with #, for example #5E6AD2.")
		return
	}
	label, err := r.client.CreateIssueLabel(ctx, labelCreateInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Linear issue label", err.Error())
		return
	}
	state := issueLabelModel{ID: types.StringValue(label.ID), Name: types.StringValue(label.Name), Color: types.StringValue(label.Color), Description: plan.Description, TeamID: plan.TeamID}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *issueLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state issueLabelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	label, err := r.client.IssueLabel(ctx, state.ID.ValueString())
	if errors.Is(err, linear.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Linear issue label", err.Error())
		return
	}
	newState := labelState(label)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *issueLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan issueLabelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !validLabelPlan(plan) {
		resp.Diagnostics.AddAttributeError(path.Root("color"), "Invalid label color", "Color must be a six-digit hexadecimal value beginning with #, for example #5E6AD2.")
		return
	}
	input := map[string]any{"name": plan.Name.ValueString(), "color": plan.Color.ValueString()}
	if plan.Description.IsNull() {
		input["description"] = nil
	} else {
		input["description"] = plan.Description.ValueString()
	}
	label, err := r.client.UpdateIssueLabel(ctx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Linear issue label", err.Error())
		return
	}
	state := issueLabelModel{ID: types.StringValue(label.ID), Name: types.StringValue(label.Name), Color: types.StringValue(label.Color), Description: plan.Description, TeamID: plan.TeamID}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *issueLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state issueLabelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteIssueLabel(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, linear.ErrNotFound) {
		resp.Diagnostics.AddError("Unable to delete Linear issue label", err.Error())
	}
}

func (r *issueLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
