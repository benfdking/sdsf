package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

type dataSourceUsers struct {
	client *slack.Client
}

func NewDataSourceUsers() datasource.DataSource {
	return &dataSourceUsers{}
}

func (d *dataSourceUsers) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dataSourceUsers) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "slack_users"
}

func (d *dataSourceUsers) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_users** resource manages multiple Slack users, allowing for the retrieval and management of user attributes.

This resource interacts with the Slack API to fetch details for a list of users.

You can optionally filter users based on their real name or email address, providing greater flexibility in retrieving specific user information.

**Required scopes**

User tokens: users:read, users:read.email (required to access the email field)
`,
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				Computed:            false,
				MarkdownDescription: "Email address match filter.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				Computed:            false,
				MarkdownDescription: "Name like filter.",
				Optional:            true,
			},
			"users": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of users returned by the filter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"color": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User's associated color.",
						},
						"deleted": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user is deleted.",
						},
						"enterprise_user": schema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "Enterprise-specific user details.",
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Unique ID of the enterprise user.",
								},
								"enterprise_id": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Enterprise ID associated with the user.",
								},
								"enterprise_name": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "Name of the enterprise the user belongs to.",
								},
								"is_admin": schema.BoolAttribute{
									Computed:            true,
									MarkdownDescription: "Indicates if the user is an admin in the enterprise.",
								},
								"is_owner": schema.BoolAttribute{
									Computed:            true,
									MarkdownDescription: "Indicates if the user is the owner in the enterprise.",
								},
								"teams": schema.ListAttribute{
									ElementType:         types.StringType,
									Computed:            true,
									MarkdownDescription: "List of team IDs the user is part of.",
								},
							},
						},
						"has_2fa": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user has 2FA enabled.",
						},
						"has_files": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user has files associated with their account.",
						},
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Unique ID of the user.",
						},
						"is_admin": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user is an admin.",
						},
						"is_app_user": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user is an app user.",
						},
						"is_bot": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user is a bot.",
						},
						"is_invited_user": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user is invited.",
						},
						"is_owner": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user is the owner.",
						},
						"is_primary_owner": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user is the primary owner.",
						},
						"is_restricted": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user has restricted access.",
						},
						"is_stranger": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user is a stranger (external user).",
						},
						"is_ultra_restricted": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates if the user has ultra-restricted access.",
						},
						"locale": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User's locale.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User's name.",
						},
						"presence": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User's presence status.",
						},
						"profile": schema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "User profile details.",
							Attributes: map[string]schema.Attribute{
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
						},
						"real_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User's real name.",
						},
						"team_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "ID of the user's team.",
						},
						"tz": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User's time zone.",
						},
						"tz_label": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Label for the user's time zone.",
						},
						"tz_offset": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Time zone offset for the user in seconds.",
						},
						"updated": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The timestamp the resource was last updated.",
						},
					},
				},
			},
		},
	}
}

func (d *dataSourceUsers) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var filterEmail, filterName types.String

	diags := func(diags diag.Diagnostics) bool {
		resp.Diagnostics.Append(diags...)
		return resp.Diagnostics.HasError()
	}

	if diags(req.Config.GetAttribute(ctx, path.Root("email"), &filterEmail)) {
		return
	}
	if diags(req.Config.GetAttribute(ctx, path.Root("name"), &filterName)) {
		return
	}

	filterByEmail := !filterEmail.IsNull() && filterEmail.ValueString() != ""
	filterByName := !filterName.IsNull() && filterName.ValueString() != ""

	users, err := d.client.GetUsers()
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving Slack users", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	var userModels []User
	foundUsers := false

	for _, user := range users {
		userProfile := user.Profile

		if filterByEmail && userProfile.Email != filterEmail.ValueString() {
			continue
		}
		if filterByName && user.Name != filterName.ValueString() {
			continue
		}

		// Mark that we have found at least one user
		foundUsers = true

		userModel := User{
			Color:   types.StringValue(user.Color),
			Deleted: types.BoolValue(user.Deleted),
			Enterprise: EnterpriseUser{
				ID:             types.StringValue(user.Enterprise.ID),
				EnterpriseID:   types.StringValue(user.Enterprise.EnterpriseID),
				EnterpriseName: types.StringValue(user.Enterprise.EnterpriseName),
				IsAdmin:        types.BoolValue(user.Enterprise.IsAdmin),
				IsOwner:        types.BoolValue(user.Enterprise.IsOwner),
				Teams: func() []types.String {
					teams := make([]types.String, len(user.Enterprise.Teams))
					for i, team := range user.Enterprise.Teams {
						teams[i] = types.StringValue(team)
					}
					return teams
				}(),
			},
			Has2FA:            types.BoolPointerValue(user.Has2FA),
			HasFiles:          types.BoolValue(user.HasFiles),
			ID:                types.StringValue(user.ID),
			IsAdmin:           types.BoolValue(user.IsAdmin),
			IsAppUser:         types.BoolValue(user.IsAppUser),
			IsBot:             types.BoolValue(user.IsBot),
			IsInvitedUser:     types.BoolValue(user.IsInvitedUser),
			IsOwner:           types.BoolValue(user.IsOwner),
			IsPrimaryOwner:    types.BoolValue(user.IsPrimaryOwner),
			IsRestricted:      types.BoolValue(user.IsRestricted),
			IsStranger:        types.BoolValue(user.IsStranger),
			IsUltraRestricted: types.BoolValue(user.IsUltraRestricted),
			Locale:            types.StringValue(user.Locale),
			Name:              types.StringValue(user.Name),
			Presence:          types.StringValue(user.Presence),
			Profile: UserProfile{
				ApiAppID:              types.StringValue(user.Profile.ApiAppID),
				AvatarHash:            types.StringValue(user.Profile.AvatarHash),
				BotID:                 types.StringValue(user.Profile.BotID),
				DisplayName:           types.StringValue(user.Profile.DisplayName),
				DisplayNameNormalized: types.StringValue(user.Profile.DisplayNameNormalized),
				Email:                 types.StringValue(user.Profile.Email),
				FirstName:             types.StringValue(user.Profile.FirstName),
				Image192:              types.StringValue(user.Profile.Image192),
				Image24:               types.StringValue(user.Profile.Image24),
				Image32:               types.StringValue(user.Profile.Image32),
				Image48:               types.StringValue(user.Profile.Image48),
				Image512:              types.StringValue(user.Profile.Image512),
				Image72:               types.StringValue(user.Profile.Image72),
				ImageOriginal:         types.StringValue(user.Profile.ImageOriginal),
				LastName:              types.StringValue(user.Profile.LastName),
				Phone:                 types.StringValue(user.Profile.Phone),
				RealName:              types.StringValue(user.Profile.RealName),
				RealNameNormalized:    types.StringValue(user.Profile.RealNameNormalized),
				Skype:                 types.StringValue(user.Profile.Skype),
				StatusEmoji:           types.StringValue(user.Profile.StatusEmoji),
				StatusExpiration:      types.Int64Value(int64(user.Profile.StatusExpiration)),
				StatusText:            types.StringValue(user.Profile.StatusText),
				Team:                  types.StringValue(user.Profile.Team),
				Title:                 types.StringValue(user.Profile.Title),
			},
			RealName: types.StringValue(user.RealName),
			TeamID:   types.StringValue(user.TeamID),
			TZ:       types.StringValue(user.TZ),
			TZLabel:  types.StringValue(user.TZLabel),
			TZOffset: types.Int64Value(int64(user.TZOffset)),
			Updated:  user.Updated,
		}

		userModels = append(userModels, userModel)
	}

	if !foundUsers {
		resp.Diagnostics.AddError("No users found", "No users were retrieved from Slack.")
		return
	}

	state := struct {
		Email types.String `tfsdk:"email"`
		Name  types.String `tfsdk:"name"`
		Users []User       `tfsdk:"users"`
	}{
		Email: filterEmail,
		Name:  filterName,
		Users: userModels,
	}

	if diags(resp.State.Set(ctx, &state)) {
		return
	}
}
