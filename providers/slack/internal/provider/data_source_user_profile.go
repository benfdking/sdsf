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

type dataSourceUserProfile struct {
	client *slack.Client
}

func NewDataSourceUserProfile() datasource.DataSource {
	return &dataSourceUserProfile{}
}

func (d *dataSourceUserProfile) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dataSourceUserProfile) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "slack_user_profile"
}

func (d *dataSourceUserProfile) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_user_profile** resource manages a specific Slack user profile by their unique user ID. It retrieves and manages user attributes to ensure consistency in user information across your infrastructure.

This resource interacts with the Slack API to fetch user details based on the specified user ID.

**Required scopes**

User tokens: users:read
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID to lookup.",
				Required:            true,
			},
			"api_app_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the associated API app.",
			},
			"avatar_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's avatar hash.",
			},
			"bot_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the associated bot.",
			},
			"display_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's display name.",
			},
			"display_name_normalized": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's normalized display name.",
			},
			"email": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's email address.",
			},
			"first_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's first name.",
			},
			"image_192": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the user's 192px image.",
			},
			"image_24": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the user's 24px image.",
			},
			"image_32": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the user's 32px image.",
			},
			"image_48": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the user's 48px image.",
			},
			"image_512": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the user's 512px image.",
			},
			"image_72": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the user's 72px image.",
			},
			"image_original": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the user's original image.",
			},
			"last_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's last name.",
			},
			"phone": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's phone number.",
			},
			"real_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's real name.",
			},
			"real_name_normalized": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's normalized real name.",
			},
			"skype": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's Skype ID.",
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
			"team": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the workspace the user is in.",
			},
			"title": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User's title or role.",
			},
		},
	}
}

func (d *dataSourceUserProfile) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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
		ID                    types.String `tfsdk:"id"`
		ApiAppID              types.String `tfsdk:"api_app_id"`
		AvatarHash            types.String `tfsdk:"avatar_hash"`
		BotID                 types.String `tfsdk:"bot_id"`
		DisplayName           types.String `tfsdk:"display_name"`
		DisplayNameNormalized types.String `tfsdk:"display_name_normalized"`
		Email                 types.String `tfsdk:"email"`
		FirstName             types.String `tfsdk:"first_name"`
		Image192              types.String `tfsdk:"image_192"`
		Image24               types.String `tfsdk:"image_24"`
		Image32               types.String `tfsdk:"image_32"`
		Image48               types.String `tfsdk:"image_48"`
		Image512              types.String `tfsdk:"image_512"`
		Image72               types.String `tfsdk:"image_72"`
		ImageOriginal         types.String `tfsdk:"image_original"`
		LastName              types.String `tfsdk:"last_name"`
		Phone                 types.String `tfsdk:"phone"`
		RealName              types.String `tfsdk:"real_name"`
		RealNameNormalized    types.String `tfsdk:"real_name_normalized"`
		Skype                 types.String `tfsdk:"skype"`
		StatusEmoji           types.String `tfsdk:"status_emoji"`
		StatusExpiration      types.Int64  `tfsdk:"status_expiration"`
		StatusText            types.String `tfsdk:"status_text"`
		Team                  types.String `tfsdk:"team"`
		Title                 types.String `tfsdk:"title"`
	}{
		ID:                    types.StringValue(foundUser.ID),
		ApiAppID:              types.StringValue(foundUser.Profile.ApiAppID),
		AvatarHash:            types.StringValue(foundUser.Profile.AvatarHash),
		BotID:                 types.StringValue(foundUser.Profile.BotID),
		DisplayName:           types.StringValue(foundUser.Profile.DisplayName),
		DisplayNameNormalized: types.StringValue(foundUser.Profile.DisplayNameNormalized),
		Email:                 types.StringValue(foundUser.Profile.Email),
		FirstName:             types.StringValue(foundUser.Profile.FirstName),
		Image192:              types.StringValue(foundUser.Profile.Image192),
		Image24:               types.StringValue(foundUser.Profile.Image24),
		Image32:               types.StringValue(foundUser.Profile.Image32),
		Image48:               types.StringValue(foundUser.Profile.Image48),
		Image512:              types.StringValue(foundUser.Profile.Image512),
		Image72:               types.StringValue(foundUser.Profile.Image72),
		ImageOriginal:         types.StringValue(foundUser.Profile.ImageOriginal),
		LastName:              types.StringValue(foundUser.Profile.LastName),
		Phone:                 types.StringValue(foundUser.Profile.Phone),
		RealName:              types.StringValue(foundUser.Profile.RealName),
		RealNameNormalized:    types.StringValue(foundUser.Profile.RealNameNormalized),
		Skype:                 types.StringValue(foundUser.Profile.Skype),
		StatusEmoji:           types.StringValue(foundUser.Profile.StatusEmoji),
		StatusExpiration:      types.Int64Value(int64(foundUser.Profile.StatusExpiration)),
		StatusText:            types.StringValue(foundUser.Profile.StatusText),
		Team:                  types.StringValue(foundUser.Profile.Team),
		Title:                 types.StringValue(foundUser.Profile.Title),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
