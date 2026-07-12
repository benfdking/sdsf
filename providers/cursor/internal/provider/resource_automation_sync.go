package provider

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/benfdking/sdsf/providers/cursor/internal/cursorautomations"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &automationSyncResource{}
	_ resource.ResourceWithConfigure = &automationSyncResource{}
)

type automationSyncResource struct {
	client *cursorautomations.Client
}

type automationSyncModel struct {
	ID                   types.String `tfsdk:"id"`
	AutomationsDirectory types.String `tfsdk:"automations_directory"`
	StatePath            types.String `tfsdk:"state_path"`
	Created              types.Int64  `tfsdk:"created"`
	Updated              types.Int64  `tfsdk:"updated"`
	Unchanged            types.Int64  `tfsdk:"unchanged"`
	LastOutput           types.String `tfsdk:"last_output"`
}

func NewAutomationSyncResource() resource.Resource { return &automationSyncResource{} }

func (r *automationSyncResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automation_sync"
}

func (r *automationSyncResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reconciles a directory of Cursor Automation definitions. Removing this resource stops management; it does not delete live Cursor automations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"automations_directory": schema.StringAttribute{
				Required:    true,
				Description: "Directory containing one subdirectory per automation, each with automation.yaml and prompt.md.",
			},
			"state_path": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Path for the reconciliation baseline cache. Defaults to .cursor-automations-state.json inside automations_directory.",
			},
			"created":     schema.Int64Attribute{Computed: true},
			"updated":     schema.Int64Attribute{Computed: true},
			"unchanged":   schema.Int64Attribute{Computed: true},
			"last_output": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *automationSyncResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *automationSyncResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan automationSyncModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *automationSyncResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan automationSyncModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *automationSyncResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state automationSyncModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

func (r *automationSyncResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
	// The copied Cursor client has no delete endpoint. Removing the Terraform
	// resource intentionally leaves live automations in place and stops syncing.
}

func (r *automationSyncResource) apply(ctx context.Context, plan *automationSyncModel, diagnostics interface {
	AddError(string, string)
}) {
	directory := plan.AutomationsDirectory.ValueString()
	statePath := plan.StatePath.ValueString()
	if statePath == "" {
		statePath = filepath.Join(directory, ".cursor-automations-state.json")
	}

	automations, err := cursorautomations.Load(directory)
	if err != nil {
		diagnostics.AddError("Unable to load Cursor automations", err.Error())
		return
	}

	var output bytes.Buffer
	result := cursorautomations.Sync(ctx, r.client, automations, cursorautomations.LoadState(statePath), statePath, &output)
	if result.Failures > 0 {
		diagnostics.AddError("Cursor automation sync failed", output.String())
		return
	}

	absDirectory, err := filepath.Abs(directory)
	if err != nil {
		diagnostics.AddError("Unable to resolve automations directory", err.Error())
		return
	}
	plan.ID = types.StringValue(absDirectory)
	plan.StatePath = types.StringValue(statePath)
	plan.Created = types.Int64Value(int64(result.Created))
	plan.Updated = types.Int64Value(int64(result.Updated))
	plan.Unchanged = types.Int64Value(int64(result.Unchanged))
	plan.LastOutput = types.StringValue(output.String())
}
