package provider

import (
	"context"
	"fmt"
	"github.com/benfdking/sdsf/providers/slack/internal/slackutil"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/slack-go/slack"
)

var (
	_ resource.Resource = (*resourceSlackUserGroup)(nil)
)

type UserGroup struct {
	AutoType    types.String        `tfsdk:"auto_type"`
	Channels    basetypes.ListValue `tfsdk:"channels"`
	CreatedBy   types.String        `tfsdk:"created_by"`
	DateCreate  types.Int64         `tfsdk:"date_create"`
	DateDelete  types.Int64         `tfsdk:"date_delete"`
	DateUpdate  types.Int64         `tfsdk:"date_update"`
	DeletedBy   types.String        `tfsdk:"deleted_by"`
	Description types.String        `tfsdk:"description"`
	Groups      basetypes.ListValue `tfsdk:"groups"`
	Handle      types.String        `tfsdk:"handle"`
	ID          types.String        `tfsdk:"id"`
	IsExternal  types.Bool          `tfsdk:"is_external"`
	IsUserGroup types.Bool          `tfsdk:"is_usergroup"`
	Name        types.String        `tfsdk:"name"`
	TeamID      types.String        `tfsdk:"team_id"`
	UpdatedBy   types.String        `tfsdk:"updated_by"`
	UserCount   types.Int64         `tfsdk:"user_count"`
	UsersId     basetypes.ListValue `tfsdk:"users_id"`
	UsersEmail  basetypes.ListValue `tfsdk:"users_email"`
}

type UserGroupSimple struct {
	AutoType    string   `json:"auto_type"`
	Channels    []string `json:"channels"`
	CreatedBy   string   `json:"created_by"`
	DateCreate  int64    `json:"date_create"`
	DateDelete  int64    `json:"date_delete"`
	DateUpdate  int64    `json:"date_update"`
	DeletedBy   string   `json:"deleted_by"`
	Description string   `json:"description"`
	Groups      []string `json:"groups"`
	Handle      string   `json:"handle"`
	ID          string   `json:"id"`
	IsExternal  bool     `json:"is_external"`
	IsUserGroup bool     `json:"is_usergroup"`
	Name        string   `json:"name"`
	TeamID      string   `json:"team_id"`
	UpdatedBy   string   `json:"updated_by"`
	UserCount   int64    `json:"user_count"`
	UsersId     []string `json:"users_id"`
	UsersEmail  []string `json:"users_email"`
}

type resourceSlackUserGroup struct {
	client *slack.Client
}

func NewResourceSlackUserGroup() resource.Resource {
	return &resourceSlackUserGroup{}
}

func (r *resourceSlackUserGroup) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		providerClient, ok := req.ProviderData.(*slackClient)
		if !ok {
			resp.Diagnostics.AddError("Invalid Provider Data", "Expected *ConfiguredClient but got something else.")
			return
		}
		r.client = providerClient.Client
	}
}

func (r *resourceSlackUserGroup) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "slack_user_group"
}

func (r *resourceSlackUserGroup) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Get attributes
	configAutoType, _ := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "auto_type", &resp.Diagnostics)
	configChannels, configChannelsIsDefined := slackutil.GetConfigAttribute[[]string](ctx, req.Config, "channels", &resp.Diagnostics)
	configDescription, _ := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "description", &resp.Diagnostics)
	configHandle, _ := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "handle", &resp.Diagnostics)
	configName, _ := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "name", &resp.Diagnostics)
	configTeamId, configTeamIdIsDefined := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "team_id", &resp.Diagnostics)

	var data UserGroup

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// fail when channel is empty
	if !data.Channels.IsNull() && len(configChannels) == 0 {
		resp.Diagnostics.AddError(
			"Usergroup Channel Validation Error on Create",
			"Channels can be null, but they cannot be defined as empty.",
		)
		return
	}

	// translate conversation names to ids
	conversationIds, err := slackutil.GetConversationIds(r.client, configChannels, []string{"public_channel", "private_channel"}, 1000)
	if err != nil {
		resp.Diagnostics.AddError(
			"Channel Retrieval Error on Create",
			fmt.Sprintf("Failed to retrieve conversation IDs: %v", err),
		)
		return
	}

	// Compute `team_id` if it’s not defined
	if !configTeamIdIsDefined {
		teamInfo, err := slackutil.GetTeamInfo(r.client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Team ID Retrieval Error",
				fmt.Sprintf("Failed to compute team ID: %v", err),
			)
			return
		}
		configTeamId = types.StringValue(teamInfo.ID)
	}

	var userGroupSimple UserGroupSimple

	userGroupInfo, _, _ := slackutil.GetUserGroupByName(r.client, configName)
	if userGroupInfo.ID == "" {
		// new
		resp.Diagnostics.AddWarning(
			"Create ...",
			fmt.Sprintln("Applying create - new"),
		)

		slackUserGroup := slack.UserGroup{
			AutoType:    configAutoType.ValueString(),
			Description: configDescription.ValueString(),
			Handle:      configHandle.ValueString(),
			Name:        configName.ValueString(),
			Prefs: slack.UserGroupPrefs{
				Channels: conversationIds,
			},
			TeamID: configTeamId.ValueString(),
		}

		userGroup, err := r.client.CreateUserGroup(slackUserGroup)
		if err != nil {
			resp.Diagnostics.AddError("Error creating Slack user group", err.Error())
			return
		}

		userGroupSimple.AutoType = userGroup.AutoType
		userGroupSimple.CreatedBy = userGroup.CreatedBy
		userGroupSimple.DateCreate = int64(userGroup.DateCreate)
		userGroupSimple.DateDelete = int64(userGroup.DateDelete)
		userGroupSimple.DateUpdate = int64(userGroup.DateUpdate)
		userGroupSimple.DeletedBy = userGroup.DeletedBy
		userGroupSimple.Description = userGroup.Description
		userGroupSimple.Handle = userGroup.Handle
		userGroupSimple.ID = userGroup.ID
		userGroupSimple.IsExternal = userGroup.IsExternal
		userGroupSimple.IsUserGroup = userGroup.IsUserGroup
		userGroupSimple.Name = userGroup.Name
		userGroupSimple.Channels = userGroup.Prefs.Channels
		userGroupSimple.Groups = userGroup.Prefs.Groups
		userGroupSimple.TeamID = userGroup.TeamID
		userGroupSimple.UpdatedBy = userGroup.UpdatedBy
		userGroupSimple.UserCount = int64(userGroup.UserCount)
		userGroupSimple.UsersId = userGroup.Users
	} else {
		// exist
		resp.Diagnostics.AddWarning(
			"Create ...",
			fmt.Sprintln("Applying create - existing resource"),
		)

		var channels []string
		// check if configChannels is defined and not empty
		if !configChannelsIsDefined || len(configChannels) == 0 {
			channels = nil
		} else {
			channels = conversationIds
		}

		// enable usergroup if its not
		if userGroupInfo.DeletedBy != "" {
			_, err := r.client.EnableUserGroup(userGroupInfo.ID)
			if err != nil {
				resp.Diagnostics.AddError("Error enabling Slack user group", err.Error())
				return
			}
		}

		options := []slack.UpdateUserGroupsOption{
			slack.UpdateUserGroupsOptionName(configName.ValueString()),
			slack.UpdateUserGroupsOptionHandle(configHandle.ValueString()),
			slack.UpdateUserGroupsOptionDescription(configDescription.ValueStringPointer()),
			slack.UpdateUserGroupsOptionChannels(channels),
		}

		userGroup, err := r.client.UpdateUserGroup(userGroupInfo.ID, options...)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Slack user group", err.Error())
			return
		}

		userGroupSimple.AutoType = configAutoType.ValueString()
		userGroupSimple.CreatedBy = userGroup.CreatedBy
		userGroupSimple.DateCreate = int64(userGroup.DateCreate)
		userGroupSimple.DateDelete = int64(userGroup.DateDelete)
		userGroupSimple.DateUpdate = int64(userGroup.DateUpdate)
		userGroupSimple.DeletedBy = userGroup.DeletedBy
		userGroupSimple.Description = configDescription.ValueString()
		userGroupSimple.Handle = configHandle.ValueString()
		userGroupSimple.ID = userGroup.ID
		userGroupSimple.IsExternal = userGroup.IsExternal
		userGroupSimple.IsUserGroup = userGroup.IsUserGroup
		userGroupSimple.Name = configName.ValueString()
		userGroupSimple.Channels = userGroup.Prefs.Channels
		userGroupSimple.Groups = userGroup.Prefs.Groups
		userGroupSimple.TeamID = configTeamId.ValueString()
		userGroupSimple.UpdatedBy = userGroup.UpdatedBy
		userGroupSimple.UserCount = int64(userGroup.UserCount)
		userGroupSimple.UsersId = userGroup.Users
	}

	// handle if empty, set to null
	if userGroupSimple.AutoType == "" {
		data.AutoType = types.StringNull()
	} else {
		data.AutoType = types.StringValue(userGroupSimple.AutoType)
	}

	// translate id to email
	usersInfo, err := slackutil.GetUserEmails(r.client, userGroupSimple.UsersId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Id lookup error",
			fmt.Sprintf("An error occurred translating user email to Id: %s", err.Error()),
		)
		return
	}

	// handle if empty, set to null
	if userGroupSimple.UserCount == 0 {
		data.UsersEmail = types.ListNull(types.StringType)
		data.UsersId = types.ListNull(types.StringType)
	} else {
		userEmailsList, err := slackutil.ConvertStringsToBasetypesList(usersInfo.Emails)
		if err != nil {
			resp.Diagnostics.AddError(
				"User Email Conversion Error",
				fmt.Sprintf("Failed to convert user emails to base types list: %s", err.Error()),
			)
			return
		}
		data.UsersEmail = userEmailsList

		userIdsList, err := slackutil.ConvertStringsToBasetypesList(userGroupSimple.UsersId)
		if err != nil {
			resp.Diagnostics.AddError(
				"User Id Conversion Error",
				fmt.Sprintf("Failed to convert user Ids to base types list: %s", err.Error()),
			)
			return
		}
		data.UsersId = userIdsList
	}

	// handle if empty, set to null
	if !configChannelsIsDefined || len(userGroupSimple.Channels) == 0 {
		data.Channels = types.ListNull(types.StringType)
	} else {
		conversationNames, err := slackutil.GetConversationNames(r.client,
			userGroupSimple.Channels,
			[]string{"public_channel", "private_channel"},
			1000)
		if err != nil {
			resp.Diagnostics.AddError(
				"Channel Retrieval Error on Create from Existing",
				fmt.Sprintf("Failed to retrieve conversation Names: %v", err),
			)
			return
		}

		userGroupChannelsList, err := slackutil.ConvertStringsToBasetypesList(conversationNames)
		if err != nil {
			resp.Diagnostics.AddError(
				"Usergroup Channels Conversion Error",
				fmt.Sprintf("Failed to convert usergroup channels to base types list: %s", err.Error()),
			)
			return
		}

		data.Channels = userGroupChannelsList
	}

	userGroupGroupsList, err := slackutil.ConvertStringsToBasetypesList(userGroupSimple.Groups)
	if err != nil {
		resp.Diagnostics.AddError(
			"Usergroup Groups Conversion Error",
			fmt.Sprintf("Failed to convert usergroup groups to base types list: %s", err.Error()),
		)
		return
	}

	data.Groups = userGroupGroupsList
	data.CreatedBy = types.StringValue(userGroupSimple.CreatedBy)
	data.DateCreate = types.Int64Value(userGroupSimple.DateCreate)
	data.DateDelete = types.Int64Value(userGroupSimple.DateDelete)
	data.DateUpdate = types.Int64Value(userGroupSimple.DateUpdate)
	data.DeletedBy = types.StringValue(userGroupSimple.DeletedBy)
	data.Description = types.StringValue(userGroupSimple.Description)
	data.Handle = types.StringValue(userGroupSimple.Handle)
	data.ID = types.StringValue(userGroupSimple.ID)
	data.IsExternal = types.BoolValue(userGroupSimple.IsExternal)
	data.IsUserGroup = types.BoolValue(userGroupSimple.IsUserGroup)
	data.Name = types.StringValue(userGroupSimple.Name)
	data.UpdatedBy = types.StringValue(userGroupSimple.UpdatedBy)
	data.UserCount = types.Int64Value(userGroupSimple.UserCount)
	data.TeamID = types.StringValue(userGroupSimple.TeamID)

	if diags := resp.State.Set(ctx, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}
}

func (r *resourceSlackUserGroup) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserGroup

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DisableUserGroup(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error disabling Slack user group", err.Error())
		return
	}

	tflog.Trace(ctx, "Disabled Slack user group", map[string]interface{}{
		"id":   data.ID.ValueString(),
		"name": data.Name.ValueString(),
	})
}

func (r *resourceSlackUserGroup) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.State.SetAttribute(ctx, path.Root("id"), req.ID)

	readReq := resource.ReadRequest{
		State: resp.State,
	}

	readResp := resource.ReadResponse{
		Diagnostics: resp.Diagnostics,
		State:       resp.State,
	}

	r.Read(ctx, readReq, &readResp)

	if readResp.Diagnostics.HasError() {
		resp.Diagnostics.Append(readResp.Diagnostics...)
		return
	}

	resp.Diagnostics.Append(readResp.Diagnostics...)
}

func (r *resourceSlackUserGroup) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	configChannels, _ := slackutil.GetConfigAttribute[[]string](ctx, req.State, "channels", &resp.Diagnostics)

	var data UserGroup

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// fail when channel is empty
	if !data.Channels.IsNull() && len(configChannels) == 0 {
		resp.Diagnostics.AddError(
			"Usergroup Channel Validation Error on Read",
			"Channels can be null, but they cannot be defined as empty.",
		)
		return
	}

	// sdk does not have a group, need to fetch all and filter
	userGroups, err := r.client.GetUserGroups(
		slack.GetUserGroupsOptionIncludeUsers(true),
		slack.GetUserGroupsOptionIncludeCount(true),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching user groups from Slack", err.Error())
		return
	}

	var userGroup *slack.UserGroup
	for _, ug := range userGroups {
		if ug.ID == data.ID.ValueString() {
			ugCopy := ug
			userGroup = &ugCopy
			break
		}
	}

	// handle if empty, set to null
	if userGroup.AutoType == "" {
		data.AutoType = types.StringNull()
	} else {
		data.AutoType = types.StringValue(userGroup.AutoType)
	}

	// translate id to email
	usersInfo, err := slackutil.GetUserEmails(r.client, userGroup.Users)
	if err != nil {
		resp.Diagnostics.AddError(
			"Id lookup error",
			fmt.Sprintf("An error occurred translating user email to Id: %s", err.Error()),
		)
		return
	}

	// handle if empty, set to null
	if userGroup.UserCount == 0 {
		data.UsersEmail = types.ListNull(types.StringType)
		data.UsersId = types.ListNull(types.StringType)
	} else {
		userEmailsList, err := slackutil.ConvertStringsToBasetypesList(usersInfo.Emails)
		if err != nil {
			resp.Diagnostics.AddError(
				"User Email Conversion Error",
				fmt.Sprintf("Failed to convert user emails to base types list: %s", err.Error()),
			)
			return
		}
		data.UsersEmail = userEmailsList

		userIdsList, err := slackutil.ConvertStringsToBasetypesList(userGroup.Users)
		if err != nil {
			resp.Diagnostics.AddError(
				"User Id Conversion Error",
				fmt.Sprintf("Failed to convert user Ids to base types list: %s", err.Error()),
			)
			return
		}
		data.UsersId = userIdsList
	}

	// handle if empty, set to null
	if len(userGroup.Prefs.Channels) > 0 {
		conversationNames, err := slackutil.GetConversationNames(r.client, userGroup.Prefs.Channels, []string{"public_channel", "private_channel"}, 1000)
		if err != nil {
			resp.Diagnostics.AddError(
				"Channel Retrieval Error on Read",
				fmt.Sprintf("Failed to retrieve conversation Names: %v", err),
			)
			return
		}

		userGroupChannelsList, err := slackutil.ConvertStringsToBasetypesList(conversationNames)
		if err != nil {
			resp.Diagnostics.AddError(
				"Usergroup Channels Conversion Error",
				fmt.Sprintf("Failed to convert usergroup channels to base types list: %s", err.Error()),
			)
			return
		}
		data.Channels = userGroupChannelsList
	}

	userGroupGroupsList, err := slackutil.ConvertStringsToBasetypesList(userGroup.Prefs.Groups)
	if err != nil {
		resp.Diagnostics.AddError(
			"Usergroup Groups Conversion Error",
			fmt.Sprintf("Failed to convert usergroup groups to base types list: %s", err.Error()),
		)
		return
	}
	data.Groups = userGroupGroupsList

	data.CreatedBy = types.StringValue(userGroup.CreatedBy)
	data.DateCreate = types.Int64Value(int64((userGroup.DateCreate)))
	data.DateDelete = types.Int64Value(int64((userGroup.DateDelete)))
	data.DateUpdate = types.Int64Value(int64((userGroup.DateUpdate)))
	data.DeletedBy = types.StringValue(userGroup.DeletedBy)
	data.Description = types.StringValue(userGroup.Description)
	data.Handle = types.StringValue(userGroup.Handle)
	data.ID = types.StringValue(userGroup.ID)
	data.IsExternal = types.BoolValue(userGroup.IsExternal)
	data.IsUserGroup = types.BoolValue(userGroup.IsUserGroup)
	data.Name = types.StringValue(userGroup.Name)
	data.TeamID = types.StringValue(userGroup.TeamID)
	data.UpdatedBy = types.StringValue(userGroup.UpdatedBy)
	data.UserCount = types.Int64Value(int64((userGroup.UserCount)))

	if diags := resp.State.Set(ctx, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceSlackUserGroup) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_user_group** resource manages Slack user groups.

This resource interacts with the Slack API to fetch user details based on the specified user ID.

- **Existing Group**: If the group name matches, it updates the existing group.
- **New Group**: If no group matches the name, it creates a new group.

**Required scopes**

User tokens: users:read, usergroups:write, team:read
`,
		Attributes: map[string]schema.Attribute{
			"auto_type": schema.StringAttribute{
				MarkdownDescription: "An optional auto type for the user group.",
				Optional:            true,
			},
			"channels": schema.ListAttribute{
				MarkdownDescription: "The preferred channels for the Slack user group.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The user who created the Slack user group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"date_create": schema.Int64Attribute{
				MarkdownDescription: "The date when the Slack user group was created.",
				Computed:            true,
			},
			"date_update": schema.Int64Attribute{
				MarkdownDescription: "The date when the Slack user group was last updated.",
				Computed:            true,
			},
			"date_delete": schema.Int64Attribute{
				MarkdownDescription: "The date when the Slack user group was deleted.",
				Computed:            true,
			},
			"deleted_by": schema.StringAttribute{
				MarkdownDescription: "The user who deleted the Slack user group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "An optional description of the Slack user group.",
				Optional:            true,
			},
			"groups": schema.ListAttribute{
				MarkdownDescription: "The preferred groups for the Slack user group.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"handle": schema.StringAttribute{
				MarkdownDescription: "The handle of the Slack user group.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The computed ID of the Slack user group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_external": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the user group is external.",
				Computed:            true,
			},
			"is_usergroup": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the user group is a Slack user group.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Slack user group.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the team associated with the Slack user group.",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_by": schema.StringAttribute{
				MarkdownDescription: "The user who last updated the Slack user group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_count": schema.Int64Attribute{
				MarkdownDescription: "The number of users in the Slack user group.",
				Computed:            true,
			},
			"users_id": schema.ListAttribute{
				MarkdownDescription: "The list of users Id in the Slack user group.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"users_email": schema.ListAttribute{
				MarkdownDescription: "The list of users email in the Slack user group.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *resourceSlackUserGroup) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get attributes
	configChannels, configChannelsIsDefined := slackutil.GetConfigAttribute[[]string](ctx, req.Config, "channels", &resp.Diagnostics)
	configDescription, _ := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "description", &resp.Diagnostics)
	configHandle, _ := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "handle", &resp.Diagnostics)
	configName, _ := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "name", &resp.Diagnostics)

	var data UserGroup

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var channels []string
	// Check if configChannels is defined and not empty
	if !configChannelsIsDefined || len(configChannels) == 0 {
		channels = nil
	} else {
		// translate conversation names to ids
		conversationIds, err := slackutil.GetConversationIds(r.client, configChannels, []string{"public_channel", "private_channel"}, 1000)
		if err != nil {
			resp.Diagnostics.AddError(
				"Channel Retrieval Error on Pre-Update",
				fmt.Sprintf("Failed to retrieve conversation IDs: %v", err),
			)
			return
		}

		channels = conversationIds
	}

	options := []slack.UpdateUserGroupsOption{
		slack.UpdateUserGroupsOptionName(configName.ValueString()),
		slack.UpdateUserGroupsOptionHandle(configHandle.ValueString()),
		slack.UpdateUserGroupsOptionDescription(configDescription.ValueStringPointer()),
		slack.UpdateUserGroupsOptionChannels(channels),
	}

	userGroup, err := r.client.UpdateUserGroup(data.ID.ValueString(), options...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Slack user group", err.Error())
		return
	}

	// handle if empty, set to null
	if userGroup.AutoType == "" {
		data.AutoType = types.StringNull()
	} else {
		data.AutoType = types.StringValue(userGroup.AutoType)
	}

	// Translate id to email
	usersInfo, err := slackutil.GetUserEmails(r.client, userGroup.Users)
	if err != nil {
		resp.Diagnostics.AddError(
			"Id lookup error",
			fmt.Sprintf("An error occurred translating user email to Id: %s", err.Error()),
		)
		return
	}

	// handle if empty, set to null
	if userGroup.UserCount == 0 {
		data.UsersEmail = types.ListNull(types.StringType)
		data.UsersId = types.ListNull(types.StringType)
	} else {
		userEmailsList, err := slackutil.ConvertStringsToBasetypesList(usersInfo.Emails)
		if err != nil {
			resp.Diagnostics.AddError(
				"User Email Conversion Error",
				fmt.Sprintf("Failed to convert user emails to base types list: %s", err.Error()),
			)
			return
		}
		data.UsersEmail = userEmailsList

		userIdsList, err := slackutil.ConvertStringsToBasetypesList(userGroup.Users)
		if err != nil {
			resp.Diagnostics.AddError(
				"User Id Conversion Error",
				fmt.Sprintf("Failed to convert user Ids to base types list: %s", err.Error()),
			)
			return
		}
		data.UsersId = userIdsList
	}

	if !configChannelsIsDefined || len(userGroup.Prefs.Channels) == 0 {
		data.Channels = types.ListNull(types.StringType)
	} else {
		conversationNames, err := slackutil.GetConversationNames(r.client, userGroup.Prefs.Channels, []string{"public_channel", "private_channel"}, 1000)
		if err != nil {
			resp.Diagnostics.AddError(
				"Channel Retrieval Error on Post-Update",
				fmt.Sprintf("Failed to retrieve conversation Names: %v", err),
			)
			return
		}

		userGroupChannelsList, err := slackutil.ConvertStringsToBasetypesList(conversationNames)
		if err != nil {
			resp.Diagnostics.AddError(
				"Usergroup Channels Conversion Error",
				fmt.Sprintf("Failed to convert usergroup channels to base types list: %s", err.Error()),
			)
			return
		}
		data.Channels = userGroupChannelsList
	}

	userGroupGroupsList, err := slackutil.ConvertStringsToBasetypesList(userGroup.Prefs.Groups)
	if err != nil {
		resp.Diagnostics.AddError(
			"Usergroup Groups Conversion Error",
			fmt.Sprintf("Failed to convert usergroup groups to base types list: %s", err.Error()),
		)
		return
	}
	data.Groups = userGroupGroupsList

	data.CreatedBy = types.StringValue(userGroup.CreatedBy)
	data.DateCreate = types.Int64Value(int64(userGroup.DateCreate))
	data.DateDelete = types.Int64Value(int64(userGroup.DateDelete))
	data.DateUpdate = types.Int64Value(int64(userGroup.DateUpdate))
	data.DeletedBy = types.StringValue(userGroup.DeletedBy)
	data.Description = types.StringValue(userGroup.Description)
	data.Handle = types.StringValue(userGroup.Handle)
	data.ID = types.StringValue(userGroup.ID)
	data.IsExternal = types.BoolValue(userGroup.IsExternal)
	data.IsUserGroup = types.BoolValue(userGroup.IsUserGroup)
	data.Name = types.StringValue(userGroup.Name)
	data.TeamID = types.StringValue(userGroup.TeamID)
	data.UpdatedBy = types.StringValue(userGroup.UpdatedBy)
	data.UserCount = types.Int64Value(int64((userGroup.UserCount)))

	if diags := resp.State.Set(ctx, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}
}
