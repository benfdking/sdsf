package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type slackProvider struct {
	version string
	client  *slackClient
}

type slackProviderModel struct {
	ApiToken types.String `tfsdk:"api_token"`
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &slackProvider{
			version: version,
		}
	}
}

func (p *slackProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "slack"
	resp.Version = p.version
}

func (p *slackProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The Slack provider enables interaction with Slack resources. Configure the provider with valid credentials to begin managing Slack resources through Terraform.
`,
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "The Slack Web API token used for authentication",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *slackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config slackProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiToken := config.ApiToken.ValueString()

	if apiToken == "" {
		apiToken = os.Getenv("SLACK_API_TOKEN")
	}

	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Slack API Token",
			"The Slack provider cannot create the API client because the Slack Web API token is missing or empty. "+
				"Please set the token value in the provider configuration or use the SLACK_API_TOKEN environment variable.",
		)
		p.client = nil
		return
	}

	p.client = &slackClient{}
	diags = p.client.Configure(ctx, apiToken, "")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	authTestResp, err := p.client.AuthTest()
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to connect to Slack API",
			fmt.Sprintf("Unable to authenticate with the provided Slack token: %s", err.Error()),
		)
		return
	}

	tflog.Info(ctx, "Slack API connection successful", map[string]any{
		"team":   authTestResp.Team,
		"user":   authTestResp.User,
		"teamID": authTestResp.TeamID,
		"userID": authTestResp.UserID,
	})

	resp.DataSourceData = p.client
	resp.ResourceData = p.client
}

func (p *slackProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewResourceSlackUserGroup,
		NewResourceSlackUserGroupMember,
		NewResourceSlackUserRealName,
		NewResourceSlackUserStatus,
	}
}

func (p *slackProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataAuthtest,
		NewdataSourceConversation,
		NewdataSourceConversations,
		NewDataSourceUser,
		NewDataSourceUserGroup,
		NewDataSourceUserGroups,
		NewDataSourceUserProfile,
		NewDataSourceUsers,
		NewDataSourceUserStatus,
	}
}

func (p *slackProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewFunctionHttpRequest,
	}
}
