package provider

import (
	"context"
	"errors"

	"github.com/benfdking/sdsf/providers/linear/internal/linear"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &teamDataSource{}
var _ datasource.DataSourceWithConfigure = &teamDataSource{}

type teamDataSource struct {
	client *linear.Client
}

type teamDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	TeamID      types.String `tfsdk:"team_id"`
	Key         types.String `tfsdk:"key"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func NewTeamDataSource() datasource.DataSource { return &teamDataSource{} }

func (d *teamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *teamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Linear team by UUID or key.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "The team's UUID."},
			"team_id":     schema.StringAttribute{Optional: true, Description: "UUID of the team to look up. Exactly one of team_id or key must be set."},
			"key":         schema.StringAttribute{Optional: true, Computed: true, Description: "Team key, such as ENG. Exactly one of team_id or key must be set."},
			"name":        schema.StringAttribute{Computed: true, Description: "Team name."},
			"description": schema.StringAttribute{Computed: true, Description: "Team description."},
		},
	}
}

func (d *teamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*clientData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "The Linear provider returned an unexpected data source configuration type.")
		return
	}
	d.client = data.client
}

func (d *teamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config teamDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hasID := !config.TeamID.IsNull() && config.TeamID.ValueString() != ""
	hasKey := !config.Key.IsNull() && config.Key.ValueString() != ""
	if hasID == hasKey {
		resp.Diagnostics.AddError("Invalid team lookup", "Configure exactly one of team_id or key.")
		return
	}

	var team *linear.Team
	var err error
	if hasID {
		team, err = d.client.Team(ctx, config.TeamID.ValueString())
	} else {
		team, err = d.client.TeamByKey(ctx, config.Key.ValueString())
	}
	if err != nil {
		if errors.Is(err, linear.ErrNotFound) {
			resp.Diagnostics.AddError("Linear team not found", "No Linear team matched the configured team_id or key.")
		} else {
			resp.Diagnostics.AddError("Unable to read Linear team", err.Error())
		}
		return
	}

	state := teamDataSourceModel{
		ID:     types.StringValue(team.ID),
		TeamID: config.TeamID,
		Key:    types.StringValue(team.Key),
		Name:   types.StringValue(team.Name),
	}
	if team.Description == nil {
		state.Description = types.StringNull()
	} else {
		state.Description = types.StringValue(*team.Description)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
