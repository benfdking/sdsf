package provider

import (
	"context"
	"os"
	"strings"

	"github.com/benfdking/sdsf/providers/linear/internal/linear"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &LinearProvider{}

type LinearProvider struct {
	version string
}

type providerModel struct {
	APIKey      types.String `tfsdk:"api_key"`
	AccessToken types.String `tfsdk:"access_token"`
	Endpoint    types.String `tfsdk:"endpoint"`
}

type clientData struct {
	client *linear.Client
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &LinearProvider{version: version} }
}

func (p *LinearProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "linear"
	resp.Version = p.version
}

func (p *LinearProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Linear workspace configuration through Linear's GraphQL API.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Linear personal API key. Defaults to LINEAR_API_KEY.",
			},
			"access_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Linear OAuth access token. Defaults to LINEAR_ACCESS_TOKEN.",
			},
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Linear GraphQL endpoint. Defaults to https://api.linear.app/graphql.",
			},
		},
	}
}

func (p *LinearProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.APIKey.IsUnknown() || config.AccessToken.IsUnknown() || config.Endpoint.IsUnknown() {
		return
	}

	apiKey := strings.TrimSpace(config.APIKey.ValueString())
	accessToken := strings.TrimSpace(config.AccessToken.ValueString())
	if apiKey == "" && accessToken == "" {
		apiKey = strings.TrimSpace(os.Getenv("LINEAR_API_KEY"))
		accessToken = strings.TrimSpace(os.Getenv("LINEAR_ACCESS_TOKEN"))
	}
	if apiKey == "" && accessToken == "" {
		resp.Diagnostics.AddError("Missing Linear credentials", "Set api_key or access_token in the provider configuration, or set LINEAR_API_KEY or LINEAR_ACCESS_TOKEN.")
		return
	}
	if apiKey != "" && accessToken != "" {
		resp.Diagnostics.AddError("Conflicting Linear credentials", "Configure only one of api_key and access_token. Personal API keys and OAuth access tokens use different Authorization header formats.")
		return
	}

	authorization := apiKey
	if accessToken != "" {
		authorization = "Bearer " + accessToken
	}
	endpoint := strings.TrimSpace(config.Endpoint.ValueString())
	data := &clientData{client: linear.NewClient(endpoint, authorization, nil)}
	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *LinearProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewIssueLabelResource, NewCustomViewResource}
}

func (p *LinearProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{NewTeamDataSource, NewProjectDataSource}
}
