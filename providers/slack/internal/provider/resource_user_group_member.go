package provider

import (
	"context"
	"fmt"
	"github.com/benfdking/sdsf/providers/slack/internal/slackutil"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

var (
	_ resource.Resource = (*resourceSlackUserRealName)(nil)
)

type resourceUserGroupMember struct {
	client *slack.Client
}

func NewResourceSlackUserGroupMember() resource.Resource {
	return &resourceUserGroupMember{}
}

func (r *resourceUserGroupMember) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		providerClient, ok := req.ProviderData.(*slackClient)
		if !ok {
			resp.Diagnostics.AddError("Invalid Provider Data", "Expected *ConfiguredClient but got something else.")
			return
		}
		r.client = providerClient.Client
	}
}

func (r *resourceUserGroupMember) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "slack_user_group_member"
}

func (r *resourceUserGroupMember) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Get attributes
	configUsergroupName, ok := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "usergroup", &resp.Diagnostics)
	if !ok {
		return
	}
	configDefaultUserEmail, ok := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "default_user", &resp.Diagnostics)
	if !ok {
		return
	}
	configUsersEmail, ok := slackutil.GetConfigAttribute[[]string](ctx, req.Config, "users", &resp.Diagnostics)
	if !ok {
		return
	}

	// Convert to []string
	defaultUserEmail := []string{configDefaultUserEmail.ValueString()}

	usersEmail := []string{}

	// Handle if empty
	if configUsersEmail != nil {
		usersEmail = configUsersEmail
	}

	// Error comparing user email
	compareUsersEmail, err := slackutil.CompareStrings(configUsersEmail, defaultUserEmail, slackutil.Any)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error comparing user email",
			fmt.Sprintf("An error occurred while comparing user emails: %s", err.Error()),
		)
		return
	}

	// If the user email comparison check returns true, add a diagnostic error and return.
	if compareUsersEmail {
		resp.Diagnostics.AddError(
			"Invalid user email",
			fmt.Sprintf("The default user email %s must not be included in the list of member user's email.", configDefaultUserEmail),
		)
		return
	}

	// Merge list, removes duplicate and ascending sort
	uniqueNewUsersEmail, err := slackutil.MergeAndValidateStrings(defaultUserEmail, usersEmail)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error merging and validating users email",
			fmt.Sprintf("An error occurred while merging users email: %s", err.Error()),
		)
		return
	}

	newUsers, err := slackutil.GetUserIds(r.client, uniqueNewUsersEmail)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Retrieving UserIds",
			fmt.Sprintf("Could not fetch attributes for user group %s: %s", configUsergroupName.ValueString(), err.Error()),
		)
		return
	}

	// Join the user IDs into a comma-separated string
	joinedUserIDs := strings.Join(newUsers.IDs, ",")

	if len(newUsers.IDs) < 1 {
		resp.Diagnostics.AddError(
			"No Users Found",
			fmt.Sprintf(
				"User group '%s' must contain at least one user.",
				configUsergroupName,
			),
		)
		return
	}

	// Fetch user group attributes
	uga, err := slackutil.GetUserGroupAttributes(r.client, configUsergroupName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting user group attributes", err.Error())
		return
	}

	// Update the user group members in Slack
	_, err = r.client.UpdateUserGroupMembers(uga.ID, joinedUserIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Slack user group members", err.Error())
		return
	}

	// Prepare the state to store updated values
	var state struct {
		UserGroup   string   `tfsdk:"usergroup"`
		DefaultUser string   `tfsdk:"default_user"`
		Users       []string `tfsdk:"users"`
	}

	// Set the state with the updated group name and users
	state.UserGroup = uga.Name
	state.DefaultUser = strings.Join(defaultUserEmail, "")
	state.Users = configUsersEmail

	// Save the state
	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *resourceUserGroupMember) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// get attributes
	configUsergroupName, ok := slackutil.GetConfigAttribute[types.String](ctx, req.State, "usergroup", &resp.Diagnostics)
	if !ok {
		return
	}
	configDefaultUserEmail, ok := slackutil.GetConfigAttribute[types.String](ctx, req.State, "default_user", &resp.Diagnostics)
	if !ok {
		return
	}

	// Convert to []string
	defaultUserEmail := []string{configDefaultUserEmail.ValueString()}

	// Get user group attributes
	uga, err := slackutil.GetUserGroupAttributes(r.client, configUsergroupName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting user group attributes", err.Error())
		return
	}

	// Get default user attributes using default user email
	defaultUser, err := slackutil.GetUserAttributes(r.client, "email", strings.Join(defaultUserEmail, ""))
	if err != nil {
		resp.Diagnostics.AddError("Error getting default user attributes", err.Error())
		return
	}

	// Update the user group members in Slack
	_, err = r.client.UpdateUserGroupMembers(uga.ID, defaultUser.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Slack user group members", err.Error())
		return
	}
}

func (r *resourceUserGroupMember) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get attributes
	configUsergroupName, ok := slackutil.GetConfigAttribute[types.String](ctx, req.State, "usergroup", &resp.Diagnostics)
	if !ok {
		return
	}
	configDefaultUserEmail, ok := slackutil.GetConfigAttribute[types.String](ctx, req.State, "default_user", &resp.Diagnostics)
	if !ok {
		return
	}

	uga, err := slackutil.GetUserGroupAttributes(r.client, configUsergroupName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Retrieving User Group Attributes",
			fmt.Sprintf("Could not fetch attributes for user group %s: %s", configUsergroupName.ValueString(), err.Error()),
		)
		return
	}

	groupAttributes, _, err := slackutil.GetUserGroupByName(r.client, configUsergroupName)
	if err != nil {
		errorMsg := "An error occurred while retrieving the user group: " + err.Error()
		fmt.Printf("API error: %s\n", errorMsg)
		resp.Diagnostics.AddError(
			"Error fetching Slack user group attribute",
			errorMsg,
		)
		return
	}

	// Convert to []string
	defaultUserEmail := []string{configDefaultUserEmail.ValueString()}

	// Get the current state
	var state struct {
		UserGroup   string   `tfsdk:"usergroup"`
		DefaultUser string   `tfsdk:"default_user"`
		Users       []string `tfsdk:"users"`
	}

	// Retrieve the state data
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert []basetypes.StringValue email to []string
	usersId, err := slackutil.ConvertBasetypesListToStrings(groupAttributes.Users)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting users email",
			fmt.Sprintf("An error occurred while converting users email to Strings: %s", err.Error()),
		)
		return
	}

	// Translate email to id
	usersInfo, err := slackutil.GetUserEmails(r.client, usersId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Id lookup error",
			fmt.Sprintf("An error occurred translating user email to Id: %s", err.Error()),
		)
		return
	}

	// Remove default user from email list
	usersEmail, err := slackutil.RemoveAndValidateStrings(usersInfo.Emails, defaultUserEmail)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing and validating users email",
			fmt.Sprintf("An error occurred while removing users email: %s", err.Error()),
		)
		return
	}

	// Handle if empty, set nil instead of []
	if len(usersEmail) == 0 {
		usersEmail = nil
	}

	// Update state
	state.UserGroup = uga.Name
	state.DefaultUser = configDefaultUserEmail.ValueString()
	state.Users = usersEmail

	// Save the updated state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *resourceUserGroupMember) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_user_group_member** resource is used to manage memberships in a Slack user group.

This resource interacts with the Slack API to add or manage users within a specified Slack user group.

**Note:** Slack does not allow a user group to have an empty list of members, so there must always be at least one user in the group.

**Required scopes**

User tokens: usergroups:write
`,
		Attributes: map[string]schema.Attribute{
			"usergroup": schema.StringAttribute{
				MarkdownDescription: "The identifier or name of the Slack user group to manage membership for.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Required: true,
			},
			"default_user": schema.StringAttribute{
				MarkdownDescription: "The default user email assigned to the specified Slack user group.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"users": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of users email to assign to the specified Slack user group.",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *resourceUserGroupMember) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get attributes
	configUsergroupName, ok := slackutil.GetConfigAttribute[types.String](ctx, req.Config, "usergroup", &resp.Diagnostics)
	if !ok {
		return
	}
	configDefaultUserEmail, ok := slackutil.GetConfigAttribute[types.String](ctx, req.State, "default_user", &resp.Diagnostics)
	if !ok {
		return
	}
	configUsersEmail, ok := slackutil.GetConfigAttribute[[]string](ctx, req.Config, "users", &resp.Diagnostics)
	if !ok {
		return
	}

	// Convert to []string
	defaultUserEmail := []string{configDefaultUserEmail.ValueString()}

	usersEmail := []string{}

	// Handle if empty
	if configUsersEmail != nil {
		usersEmail = configUsersEmail
	}

	// Merge list, removes duplicate and ascending sort
	uniqueNewUsersEmail, err := slackutil.MergeAndValidateStrings(defaultUserEmail, usersEmail)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error merging and validating users email",
			fmt.Sprintf("An error occurred while merging users email: `%s`", err.Error()),
		)
		return
	}

	// Get the current state
	var state struct {
		UserGroup   string   `tfsdk:"usergroup"`
		DefaultUser string   `tfsdk:"default_user"`
		Users       []string `tfsdk:"users"`
	}

	// Retrieve the state data
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch user group attributes
	uga, err := slackutil.GetUserGroupAttributes(r.client, configUsergroupName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Retrieving User Group Attributes",
			fmt.Sprintf("Could not fetch attributes for user group %s: %s", state.UserGroup, err.Error()),
		)
		return
	}

	// Translate user email to id
	users, err := slackutil.GetUserIds(r.client, uniqueNewUsersEmail)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Retrieving UserIds",
			fmt.Sprintf("Could not fetch attributes for user group %s: %s", state.UserGroup, err.Error()),
		)
		return
	}

	// Update the user group members in Slack
	_, err = r.client.UpdateUserGroupMembers(uga.ID, strings.Join(users.IDs, ","))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Slack user group members", err.Error())
		return
	}

	// Set the state with the updated group name and users
	state.UserGroup = configUsergroupName.ValueString()
	state.DefaultUser = configDefaultUserEmail.ValueString()
	state.Users = configUsersEmail

	// Save the state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
