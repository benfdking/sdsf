package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

type dataSourceUserStatus struct {
	client *slack.Client
}

func NewDataSourceUserStatus() datasource.DataSource {
	return &dataSourceUserStatus{}
}

func (d *dataSourceUserStatus) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		providerClient, ok := req.ProviderData.(*slackClient)
		if !ok {
			resp.Diagnostics.AddError("Invalid Provider Data",
				"Expected *ConfiguredClient but got something else.")
			return
		}
		d.client = providerClient.Client
	}
}

func (d *dataSourceUserStatus) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "slack_user_status"
}

func (d *dataSourceUserStatus) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_user_status** resource manages a specific Slack user status by their unique user ID. It retrieves and manages user status to ensure consistency in user information across your infrastructure.

This resource interacts with the Slack API to fetch user details based on the specified user ID.

**Required scopes**

User tokens: users:read
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID to lookup.",
				Required:            true,
			},
			"status_emoji": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The displayed emoji that is enabled for the Slack team.",
			},
			"status_expiration": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Expiration timestamp for the user's status.",
			},
			"status_text": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's status text.",
			},
		},
	}
}

func (d *dataSourceUserStatus) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var filterId types.String

	diags := req.Config.GetAttribute(ctx, path.Root("id"), &filterId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	users, err := d.client.GetUsers()
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving Slack user", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	if len(users) == 0 {
		resp.Diagnostics.AddError("No user found", "No user was retrieved from Slack.")
		return
	}

	var foundUser *slack.User
	for _, user := range users {
		if user.ID == filterId.ValueString() {
			userCopy := user
			foundUser = &userCopy
			break
		}
	}

	if foundUser == nil {
		resp.Diagnostics.AddError("User not found", fmt.Sprintf("No user found with ID: %s", filterId.ValueString()))
		return
	}

	state := struct {
		ID               types.String `tfsdk:"id"`
		StatusEmoji      types.String `tfsdk:"status_emoji"`
		StatusExpiration types.Int64  `tfsdk:"status_expiration"`
		StatusText       types.String `tfsdk:"status_text"`
	}{
		ID:               types.StringValue(foundUser.ID),
		StatusEmoji:      types.StringValue(foundUser.Profile.StatusEmoji),
		StatusExpiration: types.Int64Value(int64(foundUser.Profile.StatusExpiration)),
		StatusText:       types.StringValue(foundUser.Profile.StatusText),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
