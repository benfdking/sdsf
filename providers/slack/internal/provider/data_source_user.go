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

type dataSourceUser struct {
	client *slack.Client
}

func NewDataSourceUser() datasource.DataSource {
	return &dataSourceUser{}
}

func (d *dataSourceUser) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dataSourceUser) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "slack_user"
}

func (d *dataSourceUser) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_user** resource manages a specific Slack user by their unique user ID. It allows for the retrieval and management of user attributes, ensuring consistency in user information across your infrastructure.

This resource interacts with the Slack API to fetch user details based on the specified user ID.

**Required scopes**

User tokens: users:read
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID to lookup.",
				Required:            true,
			},
			"user": schema.SingleNestedAttribute{
				Computed: true,
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
	}
}

func (d *dataSourceUser) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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

	userModel := User{
		Color:   types.StringValue(foundUser.Color),
		Deleted: types.BoolValue(foundUser.Deleted),
		Enterprise: EnterpriseUser{
			ID:             types.StringValue(foundUser.Enterprise.ID),
			EnterpriseID:   types.StringValue(foundUser.Enterprise.EnterpriseID),
			EnterpriseName: types.StringValue(foundUser.Enterprise.EnterpriseName),
			IsAdmin:        types.BoolValue(foundUser.Enterprise.IsAdmin),
			IsOwner:        types.BoolValue(foundUser.Enterprise.IsOwner),
			Teams: func() []types.String {
				teams := make([]types.String, len(foundUser.Enterprise.Teams))
				for i, team := range foundUser.Enterprise.Teams {
					teams[i] = types.StringValue(team)
				}
				return teams
			}(),
		},
		Has2FA:            types.BoolPointerValue(foundUser.Has2FA),
		HasFiles:          types.BoolValue(foundUser.HasFiles),
		ID:                types.StringValue(foundUser.ID),
		IsAdmin:           types.BoolValue(foundUser.IsAdmin),
		IsAppUser:         types.BoolValue(foundUser.IsAppUser),
		IsBot:             types.BoolValue(foundUser.IsBot),
		IsInvitedUser:     types.BoolValue(foundUser.IsInvitedUser),
		IsOwner:           types.BoolValue(foundUser.IsOwner),
		IsPrimaryOwner:    types.BoolValue(foundUser.IsPrimaryOwner),
		IsRestricted:      types.BoolValue(foundUser.IsRestricted),
		IsStranger:        types.BoolValue(foundUser.IsStranger),
		IsUltraRestricted: types.BoolValue(foundUser.IsUltraRestricted),
		Locale:            types.StringValue(foundUser.Locale),
		Name:              types.StringValue(foundUser.Name),
		Presence:          types.StringValue(foundUser.Presence),
		Profile: UserProfile{
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
		},
		RealName: types.StringValue(foundUser.RealName),
		TeamID:   types.StringValue(foundUser.TeamID),
		TZ:       types.StringValue(foundUser.TZ),
		TZLabel:  types.StringValue(foundUser.TZLabel),
		TZOffset: types.Int64Value(int64(foundUser.TZOffset)),
		Updated:  foundUser.Updated,
	}
	state := struct {
		ID   types.String `tfsdk:"id"`
		User User         `tfsdk:"user"`
	}{
		ID:   types.StringValue(foundUser.ID),
		User: userModel,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
