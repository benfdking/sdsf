package provider

import (
	"context"
	"fmt"
	"github.com/benfdking/sdsf/providers/slack/internal/slackutil"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

type dataSourceUserGroup struct {
	client *slack.Client
}

func NewDataSourceUserGroup() datasource.DataSource {
	return &dataSourceUserGroup{}
}

func (d *dataSourceUserGroup) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		providerClient, ok := req.ProviderData.(*slackClient)
		if !ok {
			resp.Diagnostics.AddError("Invalid Provider Data", "Expected *ConfiguredClient but got something else.")
			return
		}
		d.client = providerClient.Client
	}
}

func (d *dataSourceUserGroup) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "slack_user_group"
}

func (d *dataSourceUserGroup) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Fetch attributes
	filterUserGroupId, ok := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "id", &resp.Diagnostics)
	if !ok {
		return
	}

	filterUserGroupName, ok := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "name", &resp.Diagnostics)
	if !ok {
		return
	}

	includeUsers, ok := slackutil.GetConfigAttribute[types.Bool](ctx, req.Config, "include_users_filter", &resp.Diagnostics)
	if !ok {
		return
	}

	includeCount, ok := slackutil.GetConfigAttribute[types.Bool](ctx, req.Config, "include_count_filter", &resp.Diagnostics)
	if !ok {
		return
	}

	includeDisabled, ok := slackutil.GetConfigAttribute[types.Bool](ctx, req.Config, "include_disabled_filter", &resp.Diagnostics)
	if !ok {
		return
	}

	teamID, ok := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "team_id_filter", &resp.Diagnostics)
	if !ok {
		return
	}

	// Validation: Ensure either id or name is set, but not both
	if !filterUserGroupId.IsNull() && !filterUserGroupName.IsNull() {
		errorMsg := "Both 'id' and 'name' cannot be set. Please specify only one."
		fmt.Printf("Validation error: %s\n", errorMsg)
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			errorMsg,
		)
		return
	}

	if filterUserGroupId.IsNull() && filterUserGroupName.IsNull() {
		errorMsg := "Either 'id' or 'name' must be set."
		fmt.Printf("Validation error: %s\n", errorMsg)
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			errorMsg,
		)
		return
	}

	// Call the Slack API to get user groups with additional options
	userGroups, err := d.client.GetUserGroups(
		slack.GetUserGroupsOptionIncludeUsers(includeUsers.ValueBool()),
		slack.GetUserGroupsOptionIncludeCount(includeCount.ValueBool()),
		slack.GetUserGroupsOptionIncludeDisabled(includeDisabled.ValueBool()),
		slack.GetUserGroupsOptionWithTeamID(teamID.ValueString()),
	)

	if err != nil {
		errorMsg := "An error occurred while retrieving the user groups: " + err.Error()
		fmt.Printf("API error: %s\n", errorMsg)
		resp.Diagnostics.AddError(
			"Error fetching Slack user groups",
			errorMsg,
		)
		return
	}

	// Check if a matching group is found
	var group *slack.UserGroup
	for _, g := range userGroups {
		if !filterUserGroupId.IsNull() && g.ID == filterUserGroupId.ValueString() {
			gCopy := g
			group = &gCopy
			break
		}
		if !filterUserGroupName.IsNull() && g.Name == filterUserGroupName.ValueString() {
			gCopy := g
			group = &gCopy
			break
		}
	}

	// If no matching group is found, fail with an error
	if group == nil {
		errorMsg := "No matching Slack user group found for the given ID or name."
		fmt.Printf("Group not found: %s\n", errorMsg)
		resp.Diagnostics.AddError(
			"User Group Not Found",
			errorMsg,
		)
		return
	}

	// Populate the state with the found user group
	channels := make([]types.String, len(group.Prefs.Channels))
	for i, channel := range group.Prefs.Channels {
		channels[i] = types.StringValue(channel)
	}

	groups := make([]types.String, len(group.Prefs.Groups))
	for i, grp := range group.Prefs.Groups {
		groups[i] = types.StringValue(grp)
	}

	users := make([]types.String, len(group.Users))
	for i, user := range group.Users {
		users[i] = types.StringValue(user)
	}

	// Define state
	state := struct {
		IncludeCountFilter    types.Bool     `tfsdk:"include_count_filter"`
		IncludeDisabledFilter types.Bool     `tfsdk:"include_disabled_filter"`
		IncludeUsersFilter    types.Bool     `tfsdk:"include_users_filter"`
		TeamIDFilter          types.String   `tfsdk:"team_id_filter"`
		AutoType              types.String   `tfsdk:"auto_type"`
		Channels              []types.String `tfsdk:"channels"`
		CreatedBy             types.String   `tfsdk:"created_by"`
		DateCreate            types.Int64    `tfsdk:"date_create"`
		DateDelete            types.Int64    `tfsdk:"date_delete"`
		DateUpdate            types.Int64    `tfsdk:"date_update"`
		DeletedBy             types.String   `tfsdk:"deleted_by"`
		Description           types.String   `tfsdk:"description"`
		Groups                []types.String `tfsdk:"groups"`
		Handle                types.String   `tfsdk:"handle"`
		ID                    types.String   `tfsdk:"id"`
		IsExternal            types.Bool     `tfsdk:"is_external"`
		IsUserGroup           types.Bool     `tfsdk:"is_user_group"`
		Name                  types.String   `tfsdk:"name"`
		TeamID                types.String   `tfsdk:"team_id"`
		UpdatedBy             types.String   `tfsdk:"updated_by"`
		UserCount             types.Int64    `tfsdk:"user_count"`
		Users                 []types.String `tfsdk:"users"`
	}{
		IncludeCountFilter:    types.BoolValue(includeUsers.ValueBool()),
		IncludeDisabledFilter: types.BoolValue(includeDisabled.ValueBool()),
		IncludeUsersFilter:    types.BoolValue(includeUsers.ValueBool()),
		TeamIDFilter:          types.StringValue(teamID.ValueString()),
		AutoType:              types.StringValue(group.AutoType),
		Channels:              channels,
		CreatedBy:             types.StringValue(group.CreatedBy),
		DateCreate:            types.Int64Value(int64(group.DateCreate)),
		DateDelete:            types.Int64Value(int64(group.DateDelete)),
		DateUpdate:            types.Int64Value(int64(group.DateUpdate)),
		DeletedBy:             types.StringValue(group.DeletedBy),
		Description:           types.StringValue(group.Description),
		Groups:                groups,
		Handle:                types.StringValue(group.Handle),
		ID:                    types.StringValue(group.ID),
		IsExternal:            types.BoolValue(group.IsExternal),
		IsUserGroup:           types.BoolValue(group.IsUserGroup),
		Name:                  types.StringValue(group.Name),
		TeamID:                types.StringValue(group.TeamID),
		UpdatedBy:             types.StringValue(group.UpdatedBy),
		UserCount:             types.Int64Value(int64(group.UserCount)),
		Users:                 users,
	}

	// Set the state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *dataSourceUserGroup) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_user_group** data source retrieves information about a specific user group within Slack. It allows users to fetch details such as user group members and attributes.

You can customize the data retrieval by specifying various optional parameters:
- **Include users**: Choose whether to include user details within the group.
- **Include count**: Specify whether to return the count of users in the group.
- **Include disabled groups**: Optionally retrieve disabled user groups.
- **Team ID**: Filter groups by team ID for multi-team Slack organizations.

This flexibility enables targeted queries based on your specific needs.

**Required scopes**

- User tokens: user groups:read
`,
		Attributes: map[string]schema.Attribute{
			"include_count_filter": schema.BoolAttribute{
				MarkdownDescription: "Include the count of users in each group.",
				Optional:            true,
			},
			"include_disabled_filter": schema.BoolAttribute{
				MarkdownDescription: "Include disabled groups.",
				Optional:            true,
			},
			"include_users_filter": schema.BoolAttribute{
				MarkdownDescription: "Include users in each group.",
				Optional:            true,
			},
			"team_id_filter": schema.StringAttribute{
				MarkdownDescription: "Encoded team id to list user groups in, required if org token is used.",
				Optional:            true,
			},
			"auto_type": schema.StringAttribute{
				MarkdownDescription: "Automation type for the group, if any.",
				Computed:            true,
			},
			"channels": schema.ListAttribute{
				MarkdownDescription: "The preferred channels for the Slack user group.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "The user who created the Slack user group.",
				Computed:            true,
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
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the group.",
				Computed:            true,
			},
			"groups": schema.ListAttribute{
				MarkdownDescription: "The preferred groups for the Slack user group.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"handle": schema.StringAttribute{
				MarkdownDescription: "Short handle for mentioning the group.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the user group.",
				Optional:            true,
			},
			"is_external": schema.BoolAttribute{
				MarkdownDescription: "True if the group is external.",
				Computed:            true,
			},
			"is_user_group": schema.BoolAttribute{
				MarkdownDescription: "True if it is a user group.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the user group.",
				Optional:            true,
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "Team ID of the user group.",
				Optional:            true,
			},
			"updated_by": schema.StringAttribute{
				MarkdownDescription: "User who last updated the group.",
				Computed:            true,
			},
			"user_count": schema.Int64Attribute{
				MarkdownDescription: "Number of users in the group.",
				Computed:            true,
			},
			"users": schema.ListAttribute{
				MarkdownDescription: "List of users in the group.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}
