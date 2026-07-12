package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

type AuthTestModel struct {
	Team   types.String `tfsdk:"team"`
	User   types.String `tfsdk:"user"`
	TeamID types.String `tfsdk:"team_id"`
	UserID types.String `tfsdk:"user_id"`
}

type dataSourceAuthtest struct {
	client *slack.Client
}

func NewDataAuthtest() datasource.DataSource {
	return &dataSourceAuthtest{}
}

func (d *dataSourceAuthtest) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		providerClient, ok := req.ProviderData.(*slackClient)
		if !ok {
			resp.Diagnostics.AddError("Invalid Provider Data", "Expected *ConfiguredClient but got something else.")
			return
		}
		d.client = providerClient.Client
	}
}

func (d *dataSourceAuthtest) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "slack_authtest"
}

func (d *dataSourceAuthtest) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_authtest** data source is used to verify that a given Slack API token is valid and can successfully connect to the Slack API.

Using this data source helps ensure that your integration with Slack is set up correctly before making further API calls.
`,
		Attributes: map[string]schema.Attribute{
			"team": schema.StringAttribute{
				MarkdownDescription: "Team name associated with the Slack API token",
				Computed:            true,
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "Authenticated user name",
				Computed:            true,
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "Team ID associated with the Slack API token",
				Computed:            true,
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "Authenticated user ID",
				Computed:            true,
			},
		},
	}
}

func (d *dataSourceAuthtest) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	authTestResp, err := d.client.AuthTest()
	if err != nil {
		resp.Diagnostics.AddError("Slack API AuthTest failed", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	authTestData := AuthTestModel{
		Team:   types.StringValue(authTestResp.Team),
		User:   types.StringValue(authTestResp.User),
		TeamID: types.StringValue(authTestResp.TeamID),
		UserID: types.StringValue(authTestResp.UserID),
	}

	diags := resp.State.Set(ctx, &authTestData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
