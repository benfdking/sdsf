package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/slack-go/slack"
)

var (
	_ resource.Resource = (*resourceSlackUserStatus)(nil)
)

type resourceSlackUserStatus struct {
	client *slack.Client
}

type UserStatus struct {
	ID               types.String `tfsdk:"id"`
	StatusEmoji      types.String `tfsdk:"status_emoji"`
	StatusExpiration types.Int64  `tfsdk:"status_expiration"`
	StatusText       types.String `tfsdk:"status_text"`
}

func NewResourceSlackUserStatus() resource.Resource {
	return &resourceSlackUserStatus{}
}

func (r *resourceSlackUserStatus) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		providerClient, ok := req.ProviderData.(*slackClient)
		if !ok {
			resp.Diagnostics.AddError("Invalid Provider Data", "Expected *ConfiguredClient but got something else.")
			return
		}
		r.client = providerClient.Client
	}
}

func (r *resourceSlackUserStatus) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "slack_user_status"
}

func (r *resourceSlackUserStatus) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserStatus

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.SetUserCustomStatus(
		data.StatusText.ValueString(),
		data.StatusEmoji.ValueString(),
		data.StatusExpiration.ValueInt64(),
	)
	if err != nil {
		if slackError, ok := err.(*slack.SlackErrorResponse); ok && slackError.Err == "user_not_found" {
			resp.Diagnostics.AddError("User not found", "The specified Slack user does not exist.")
			return
		}
		resp.Diagnostics.AddError("Error setting Slack user status", err.Error())
		return
	}

	if diags := resp.State.Set(ctx, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

	tflog.Trace(ctx, "Slack user status set", map[string]interface{}{
		"id":                data.ID.ValueString(),
		"status_text":       data.StatusText.ValueString(),
		"status_emoji":      data.StatusEmoji.ValueString(),
		"status_expiration": data.StatusExpiration.ValueInt64(),
	})
}

func (r *resourceSlackUserStatus) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserStatus

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UnsetUserCustomStatus()
	if err != nil {
		if slackError, ok := err.(*slack.SlackErrorResponse); ok && slackError.Err == "user_not_found" {
			tflog.Warn(ctx, "Slack user not found, assuming it was already deleted", map[string]interface{}{
				"id": data.ID.ValueString(),
			})
		} else {
			resp.Diagnostics.AddError("Error clearing Slack user status", err.Error())
			return
		}
	}

	tflog.Trace(ctx, "Slack user status cleared", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *resourceSlackUserStatus) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserStatus

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	slackUser, err := r.client.GetUserProfile(&slack.GetUserProfileParameters{UserID: data.ID.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving Slack user status", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	if slackUser == nil {
		resp.Diagnostics.AddError("No user found", "No user was retrieved from Slack.")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &UserStatus{
		ID:               types.StringValue(data.ID.ValueString()),
		StatusText:       types.StringValue(slackUser.StatusText),
		StatusEmoji:      types.StringValue(slackUser.StatusEmoji),
		StatusExpiration: types.Int64Value(int64(slackUser.StatusExpiration)),
	})...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Read Slack user status", map[string]interface{}{
		"id":                data.ID.ValueString(),
		"status_text":       slackUser.StatusText,
		"status_emoji":      slackUser.StatusEmoji,
		"status_expiration": slackUser.StatusExpiration,
	})
}

func (r *resourceSlackUserStatus) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_user_status** manage a Slack user status details.

This resource interacts with the Slack API to fetch user details based on the specified user ID.

**Required scopes**

User tokens: users:read, users.profile:write
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Slack user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status_text": schema.StringAttribute{
				MarkdownDescription: "The text to display as the user's status.",
				Optional:            true,
			},
			"status_emoji": schema.StringAttribute{
				MarkdownDescription: "The emoji to display as the user's status.",
				Optional:            true,
			},
			"status_expiration": schema.Int64Attribute{
				MarkdownDescription: "The timestamp (epoch) when the status will expire.",
				Optional:            true,
			},
		},
	}
}

func (r *resourceSlackUserStatus) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserStatus

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.SetUserCustomStatus(
		data.StatusText.ValueString(),
		data.StatusEmoji.ValueString(),
		data.StatusExpiration.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Slack user status", err.Error())
		return
	}

	tflog.Trace(ctx, "Slack user status updated", map[string]interface{}{
		"id":                data.ID.ValueString(),
		"status_text":       data.StatusText.ValueString(),
		"status_emoji":      data.StatusEmoji.ValueString(),
		"status_expiration": data.StatusExpiration.ValueInt64(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
