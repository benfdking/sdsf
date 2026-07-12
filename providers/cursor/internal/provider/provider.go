package provider

import (
	"context"
	"os"

	"github.com/benfdking/sdsf/providers/cursor/internal/cursorautomations"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &CursorProvider{}

type CursorProvider struct {
	version string
}

type providerModel struct {
	SessionToken types.String `tfsdk:"session_token"`
	TeamID       types.String `tfsdk:"team_id"`
}

type clientData struct {
	client *cursorautomations.Client
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &CursorProvider{version: version} }
}

func (p *CursorProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cursor"
	resp.Version = p.version
}

func (p *CursorProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Cursor Automations through Cursor's dashboard API.",
		Attributes: map[string]schema.Attribute{
			"session_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Cursor WorkosCursorSessionToken cookie. Defaults to CURSOR_SESSION_TOKEN.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Description: "Cursor team ID. Defaults to CURSOR_TEAM_ID.",
			},
		},
	}
}

func (p *CursorProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionToken := config.SessionToken.ValueString()
	if sessionToken == "" {
		sessionToken = os.Getenv("CURSOR_SESSION_TOKEN")
	}
	teamID := config.TeamID.ValueString()
	if teamID == "" {
		teamID = os.Getenv("CURSOR_TEAM_ID")
	}
	if sessionToken == "" || teamID == "" {
		resp.Diagnostics.AddError(
			"Missing Cursor credentials",
			"Set session_token and team_id in the provider configuration, or set CURSOR_SESSION_TOKEN and CURSOR_TEAM_ID.",
		)
		return
	}

	data := &clientData{client: cursorautomations.NewClient(sessionToken, teamID)}
	resp.ResourceData = data
}

func (p *CursorProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewAutomationSyncResource}
}

func (p *CursorProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}
