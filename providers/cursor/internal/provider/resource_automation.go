package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/benfdking/sdsf/providers/cursor/internal/cursorautomations"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &automationResource{}
	_ resource.ResourceWithConfigure   = &automationResource{}
	_ resource.ResourceWithImportState = &automationResource{}
)

type automationResource struct {
	client cursorautomations.API
}

type automationModel struct {
	ID                    types.String  `tfsdk:"id"`
	Name                  types.String  `tfsdk:"name"`
	Scope                 types.String  `tfsdk:"scope"`
	Model                 types.String  `tfsdk:"model"`
	Prompt                types.String  `tfsdk:"prompt"`
	Enabled               types.Bool    `tfsdk:"enabled"`
	MemoryEnabled         types.Bool    `tfsdk:"memory_enabled"`
	Triggers              types.Dynamic `tfsdk:"triggers"`
	Actions               types.Dynamic `tfsdk:"actions"`
	SlackNotifiedChannels types.Set     `tfsdk:"slack_notified_channels"`
	GitConfig             types.Object  `tfsdk:"git_config"`
}

type gitConfigModel struct {
	Repositories types.Set    `tfsdk:"repositories"`
	Branch       types.String `tfsdk:"branch"`
}

func NewAutomationResource() resource.Resource { return &automationResource{} }

func (r *automationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automation"
}

func (r *automationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages one Cursor Automation. Trigger and action payloads are native HCL values using Cursor's API field names.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Cursor's server-assigned automation ID.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name. Existing automations are adopted by name during creation.",
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("TEAM_VISIBLE"),
				Description: "Cursor automation scope, in short or AUTOMATION_SCOPE_* form.",
			},
			"model": schema.StringAttribute{
				Required:    true,
				Description: "Cursor model identifier.",
			},
			"prompt": schema.StringAttribute{
				Required:    true,
				Description: "Instructions supplied to the automation agent.",
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"memory_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"triggers": schema.DynamicAttribute{
				Optional:    true,
				Description: "List of Cursor trigger objects, expressed as native HCL values (for example [{ cron = { cron = \"0 9 * * 1-5\" } }]).",
			},
			"actions": schema.DynamicAttribute{
				Optional:    true,
				Description: "List of Cursor action objects, expressed as native HCL values (for example [{ requestReviewers = {} }]).",
			},
			"slack_notified_channels": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"git_config": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"repositories": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
					},
					"branch": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("main"),
					},
				},
			},
		},
	}
}

func (r *automationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*clientData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *clientData, got %T.", req.ProviderData))
		return
	}
	r.client = data.client
}

func (r *automationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan automationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &plan, "", &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *automationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state automationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &plan, state.ID.ValueString(), &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *automationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state automationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	live, err := r.list(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Cursor automation", err.Error())
		return
	}
	for _, automation := range live {
		if automation.AutomationID == state.ID.ValueString() {
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *automationResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
	// Cursor does not publish an Automations management API, and the copied
	// dashboard client exposes no verified delete operation. Terraform therefore
	// forgets the resource without deleting the live automation.
}

func (r *automationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *automationResource) apply(ctx context.Context, plan *automationModel, preferredID string, diagnostics *diag.Diagnostics) {
	automation, buildDiags := buildAutomation(ctx, *plan)
	diagnostics.Append(buildDiags...)
	if diagnostics.HasError() {
		return
	}

	live, err := r.list(ctx)
	if err != nil {
		diagnostics.AddError("Unable to list Cursor automations", err.Error())
		return
	}

	id := ""
	for _, candidate := range live {
		if candidate.AutomationID == preferredID || (preferredID == "" && candidate.Name == automation.Config.Name) {
			id = candidate.AutomationID
			break
		}
	}
	if id == "" {
		id, err = r.client.Create(ctx, automation)
	} else {
		err = r.client.Update(ctx, automation, id)
	}
	if err != nil {
		diagnostics.AddError("Unable to apply Cursor automation", err.Error())
		return
	}

	postWrite, err := r.list(ctx)
	if err != nil {
		diagnostics.AddError("Unable to verify Cursor automation", err.Error())
		return
	}
	for _, candidate := range postWrite {
		if candidate.AutomationID != id {
			continue
		}
		plan.ID = types.StringValue(id)
		return
	}
	diagnostics.AddError("Unable to verify Cursor automation", fmt.Sprintf("Automation %q (%s) was absent from the post-write list.", automation.Config.Name, id))
}

func (r *automationResource) list(ctx context.Context) ([]cursorautomations.LiveAutomation, error) {
	raw, err := r.client.List(ctx)
	if err != nil {
		return nil, err
	}
	return cursorautomations.ParseList(raw)
}

func buildAutomation(ctx context.Context, plan automationModel) (cursorautomations.Automation, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	triggers, err := dynamicObjectList(plan.Triggers)
	if err != nil {
		diagnostics.AddError("Invalid triggers", err.Error())
	}
	actions, err := dynamicObjectList(plan.Actions)
	if err != nil {
		diagnostics.AddError("Invalid actions", err.Error())
	}

	var channels []string
	if !plan.SlackNotifiedChannels.IsNull() && !plan.SlackNotifiedChannels.IsUnknown() {
		diagnostics.Append(plan.SlackNotifiedChannels.ElementsAs(ctx, &channels, false)...)
	}

	var gitConfig *cursorautomations.GitConfig
	if !plan.GitConfig.IsNull() && !plan.GitConfig.IsUnknown() {
		var value gitConfigModel
		diagnostics.Append(plan.GitConfig.As(ctx, &value, basetypes.ObjectAsOptions{})...)
		var repositories []string
		diagnostics.Append(value.Repositories.ElementsAs(ctx, &repositories, false)...)
		gitConfig = &cursorautomations.GitConfig{Repos: repositories, Branch: value.Branch.ValueString()}
	}
	if diagnostics.HasError() {
		return cursorautomations.Automation{}, diagnostics
	}

	enabled := plan.Enabled.ValueBool()
	memoryEnabled := plan.MemoryEnabled.ValueBool()
	automation, err := cursorautomations.NewAutomation(cursorautomations.Config{
		Name:                  plan.Name.ValueString(),
		Scope:                 plan.Scope.ValueString(),
		Model:                 plan.Model.ValueString(),
		Triggers:              triggers,
		Actions:               actions,
		Enabled:               &enabled,
		MemoryEnabled:         &memoryEnabled,
		SlackNotifiedChannels: channels,
		GitConfig:             gitConfig,
	}, plan.Prompt.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid Cursor automation", err.Error())
	}
	return automation, diagnostics
}

func dynamicObjectList(value types.Dynamic) ([]map[string]any, error) {
	if value.IsNull() {
		return nil, nil
	}
	if value.IsUnknown() || value.IsUnderlyingValueUnknown() {
		return nil, fmt.Errorf("value must be known during apply")
	}
	converted, err := terraformValueToGo(value.UnderlyingValue())
	if err != nil {
		return nil, err
	}
	items, ok := converted.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list or tuple, got %T", converted)
	}
	result := make([]map[string]any, len(items))
	for i, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("item %d must be an object, got %T", i, item)
		}
		result[i] = object
	}
	return result, nil
}

func terraformValueToGo(value attr.Value) (any, error) {
	if value == nil || value.IsNull() {
		return nil, nil
	}
	if value.IsUnknown() {
		return nil, fmt.Errorf("nested value is unknown")
	}

	switch value := value.(type) {
	case basetypes.DynamicValue:
		return terraformValueToGo(value.UnderlyingValue())
	case basetypes.StringValue:
		return value.ValueString(), nil
	case basetypes.BoolValue:
		return value.ValueBool(), nil
	case basetypes.Int64Value:
		return value.ValueInt64(), nil
	case basetypes.Float64Value:
		return value.ValueFloat64(), nil
	case basetypes.NumberValue:
		return bigFloatToJSONNumber(value.ValueBigFloat()), nil
	case basetypes.ListValue:
		return terraformElementsToGo(value.Elements())
	case basetypes.SetValue:
		return terraformElementsToGo(value.Elements())
	case basetypes.TupleValue:
		return terraformElementsToGo(value.Elements())
	case basetypes.MapValue:
		return terraformAttributesToGo(value.Elements())
	case basetypes.ObjectValue:
		return terraformAttributesToGo(value.Attributes())
	default:
		return nil, fmt.Errorf("unsupported Terraform value type %T", value)
	}
}

func terraformElementsToGo(elements []attr.Value) ([]any, error) {
	result := make([]any, len(elements))
	for i, element := range elements {
		value, err := terraformValueToGo(element)
		if err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
		result[i] = value
	}
	return result, nil
}

func terraformAttributesToGo(attributes map[string]attr.Value) (map[string]any, error) {
	result := make(map[string]any, len(attributes))
	for key, attribute := range attributes {
		value, err := terraformValueToGo(attribute)
		if err != nil {
			return nil, fmt.Errorf("attribute %q: %w", key, err)
		}
		result[key] = value
	}
	return result, nil
}

func bigFloatToJSONNumber(value *big.Float) json.Number {
	return json.Number(value.Text('f', -1))
}
