package provider

import (
	"context"
	"github.com/benfdking/sdsf/providers/slack/internal/slackutil"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

type dataSourceUserGroups struct {
	client *slack.Client
}

func NewDataSourceUserGroups() datasource.DataSource {
	return &dataSourceUserGroups{}
}

func (d *dataSourceUserGroups) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		providerClient, ok := req.ProviderData.(*slackClient)
		if !ok {
			resp.Diagnostics.AddError("Invalid Provider Data", "Expected *ConfiguredClient but got something else.")
			return
		}
		d.client = providerClient.Client
	}
}

func (d *dataSourceUserGroups) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "slack_user_groups"
}

func (d *dataSourceUserGroups) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	diags := func(diags diag.Diagnostics) bool {
		resp.Diagnostics.Append(diags...)
		return resp.Diagnostics.HasError()
	}

	includeUsers, _ := slackutil.GetConfigAttribute[types.Bool](ctx, req.Config, "include_users_filter", &resp.Diagnostics)
	includeCount, _ := slackutil.GetConfigAttribute[types.Bool](ctx, req.Config, "include_count_filter", &resp.Diagnostics)
	includeDisabled, _ := slackutil.GetConfigAttribute[types.Bool](ctx, req.Config, "include_disabled_filter", &resp.Diagnostics)
	teamID, _ := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "team_id_filter", &resp.Diagnostics)

	userGroups, err := d.client.GetUserGroups(
		slack.GetUserGroupsOptionIncludeUsers(includeUsers.ValueBool()),
		slack.GetUserGroupsOptionIncludeCount(includeCount.ValueBool()),
		slack.GetUserGroupsOptionIncludeDisabled(includeDisabled.ValueBool()),
		slack.GetUserGroupsOptionWithTeamID(teamID.ValueString()),
	)

	if err != nil {
		resp.Diagnostics.AddError("Error fetching user groups", err.Error())
		return
	}

	userGroupsModels := []struct {
		ID          types.String   `tfsdk:"id"`
		TeamID      types.String   `tfsdk:"team_id"`
		IsUserGroup types.Bool     `tfsdk:"is_user_group"`
		Name        types.String   `tfsdk:"name"`
		CreatedBy   types.String   `tfsdk:"created_by"`
		DateCreate  types.Int64    `tfsdk:"date_create"`
		DateDelete  types.Int64    `tfsdk:"date_delete"`
		DateUpdate  types.Int64    `tfsdk:"date_update"`
		DeletedBy   types.String   `tfsdk:"deleted_by"`
		Description types.String   `tfsdk:"description"`
		Handle      types.String   `tfsdk:"handle"`
		UserCount   types.Int64    `tfsdk:"user_count"`
		UpdatedBy   types.String   `tfsdk:"updated_by"`
		AutoType    types.String   `tfsdk:"auto_type"`
		IsExternal  types.Bool     `tfsdk:"is_external"`
		Channels    []types.String `tfsdk:"channels"`
		Groups      []types.String `tfsdk:"groups"`
		Users       []types.String `tfsdk:"users"`
	}{}

	for _, group := range userGroups {
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
		userGroupsModels = append(userGroupsModels, struct {
			ID          types.String   `tfsdk:"id"`
			TeamID      types.String   `tfsdk:"team_id"`
			IsUserGroup types.Bool     `tfsdk:"is_user_group"`
			Name        types.String   `tfsdk:"name"`
			CreatedBy   types.String   `tfsdk:"created_by"`
			DateCreate  types.Int64    `tfsdk:"date_create"`
			DateDelete  types.Int64    `tfsdk:"date_delete"`
			DateUpdate  types.Int64    `tfsdk:"date_update"`
			DeletedBy   types.String   `tfsdk:"deleted_by"`
			Description types.String   `tfsdk:"description"`
			Handle      types.String   `tfsdk:"handle"`
			UserCount   types.Int64    `tfsdk:"user_count"`
			UpdatedBy   types.String   `tfsdk:"updated_by"`
			AutoType    types.String   `tfsdk:"auto_type"`
			IsExternal  types.Bool     `tfsdk:"is_external"`
			Channels    []types.String `tfsdk:"channels"`
			Groups      []types.String `tfsdk:"groups"`
			Users       []types.String `tfsdk:"users"`
		}{
			ID:          types.StringValue(group.ID),
			TeamID:      types.StringValue(group.TeamID),
			IsUserGroup: types.BoolValue(group.IsUserGroup),
			Name:        types.StringValue(group.Name),
			CreatedBy:   types.StringValue(group.CreatedBy),
			DateCreate:  types.Int64Value(int64(group.DateCreate)),
			DateDelete:  types.Int64Value(int64(group.DateDelete)),
			DateUpdate:  types.Int64Value(int64(group.DateUpdate)),
			DeletedBy:   types.StringValue(group.DeletedBy),
			Description: types.StringValue(group.Description),
			Handle:      types.StringValue(group.Handle),
			UserCount:   types.Int64Value(int64(group.UserCount)),
			UpdatedBy:   types.StringValue(group.UpdatedBy),
			AutoType:    types.StringValue(group.AutoType),
			IsExternal:  types.BoolValue(group.IsExternal),
			Channels:    channels,
			Groups:      groups,
			Users:       users,
		})
	}

	state := struct {
		IncludeUsersFilter    types.Bool   `tfsdk:"include_users_filter"`
		IncludeCountFilter    types.Bool   `tfsdk:"include_count_filter"`
		IncludeDisabledFilter types.Bool   `tfsdk:"include_disabled_filter"`
		TeamIDFilter          types.String `tfsdk:"team_id_filter"`
		UserGroups            []struct {
			ID          types.String   `tfsdk:"id"`
			TeamID      types.String   `tfsdk:"team_id"`
			IsUserGroup types.Bool     `tfsdk:"is_user_group"`
			Name        types.String   `tfsdk:"name"`
			CreatedBy   types.String   `tfsdk:"created_by"`
			DateCreate  types.Int64    `tfsdk:"date_create"`
			DateDelete  types.Int64    `tfsdk:"date_delete"`
			DateUpdate  types.Int64    `tfsdk:"date_update"`
			DeletedBy   types.String   `tfsdk:"deleted_by"`
			Description types.String   `tfsdk:"description"`
			Handle      types.String   `tfsdk:"handle"`
			UserCount   types.Int64    `tfsdk:"user_count"`
			UpdatedBy   types.String   `tfsdk:"updated_by"`
			AutoType    types.String   `tfsdk:"auto_type"`
			IsExternal  types.Bool     `tfsdk:"is_external"`
			Channels    []types.String `tfsdk:"channels"`
			Groups      []types.String `tfsdk:"groups"`
			Users       []types.String `tfsdk:"users"`
		} `tfsdk:"user_groups"`
	}{
		IncludeUsersFilter:    types.BoolValue(includeUsers.ValueBool()),
		IncludeCountFilter:    types.BoolValue(includeCount.ValueBool()),
		IncludeDisabledFilter: types.BoolValue(includeDisabled.ValueBool()),
		TeamIDFilter:          types.StringValue(teamID.ValueString()),
		UserGroups:            userGroupsModels,
	}

	if diags(resp.State.Set(ctx, &state)) {
		return
	}
}

func (d *dataSourceUserGroups) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_user_groups** data source retrieves information about user groups within Slack. It allows users to fetch details of multiple user groups, including their members and attributes.

You can customize the data retrieval using several optional parameters, such as whether to include user details, the count of users in each group, and whether to include disabled groups. This flexibility enables targeted queries based on your needs.

**Required scopes**

User tokens: usergroups:read
`,
		Attributes: map[string]schema.Attribute{
			"include_count_filter": schema.BoolAttribute{
				Optional:    true,
				Description: "Include the count of users in each group. Defaults to `false`.",
			},
			"include_disabled_filter": schema.BoolAttribute{
				Optional:    true,
				Description: "Include disabled groups. Defaults to `false`.",
			},
			"include_users_filter": schema.BoolAttribute{
				Optional:    true,
				Description: "Include users in each group. Defaults to `false`.",
			},
			"team_id_filter": schema.StringAttribute{
				Optional:    true,
				Description: "Encoded team id to list user groups in, required if org token is used.",
			},
			"user_groups": schema.ListNestedAttribute{
				Description: "List of Slack user groups.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
							Computed:            true,
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
							MarkdownDescription: "Name of the group.",
							Computed:            true,
						},
						"team_id": schema.StringAttribute{
							MarkdownDescription: "ID of the team (Slack workspace).",
							Computed:            true,
						},
						"updated_by": schema.StringAttribute{
							MarkdownDescription: "The user who last updated the Slack user group.",
							Computed:            true,
						},
						"user_count": schema.Int64Attribute{
							MarkdownDescription: "Number of users in the group.",
							Computed:            true,
						},
						"users": schema.ListAttribute{
							MarkdownDescription: "The list of users in the Slack usergroup.",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}
