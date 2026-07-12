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
	_ resource.Resource = (*resourceSlackUserRealName)(nil)
)

type resourceSlackUserRealName struct {
	client *slack.Client
}

type UserRealName struct {
	ID       types.String `tfsdk:"id"`
	RealName types.String `tfsdk:"real_name"`
}

func NewResourceSlackUserRealName() resource.Resource {
	return &resourceSlackUserRealName{}
}

func (r *resourceSlackUserRealName) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		providerClient, ok := req.ProviderData.(*slackClient)
		if !ok {
			resp.Diagnostics.AddError("Invalid Provider Data", "Expected *ConfiguredClient but got something else.")
			return
		}
		r.client = providerClient.Client
	}
}

func (r *resourceSlackUserRealName) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "slack_user_real_name"
}

func (r *resourceSlackUserRealName) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserRealName

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.SetUserRealNameContextWithUser(ctx, data.ID.ValueString(), data.RealName.ValueString())
	if err != nil {
		if slackError, ok := err.(*slack.SlackErrorResponse); ok && slackError.Err == "user_not_found" {
			resp.Diagnostics.AddError("User Not Found", "The specified Slack user does not exist.")
			return
		}
		resp.Diagnostics.AddError("Error Setting Slack User Real Name", err.Error())
		return
	}

	if diags := resp.State.Set(ctx, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

	tflog.Trace(ctx, "Slack user real name set", map[string]interface{}{
		"id":        data.ID.ValueString(),
		"real_name": data.RealName.ValueString(),
	})
}

func (r *resourceSlackUserRealName) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserRealName

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Warn(ctx, "Slack user real name deletion is not explicitly supported", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *resourceSlackUserRealName) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserRealName

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	slackUser, err := r.client.GetUserProfile(&slack.GetUserProfileParameters{UserID: data.ID.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error Retrieving Slack User Real Name", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	if slackUser == nil {
		resp.Diagnostics.AddError("No User Found", "No user was retrieved from Slack.")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &UserRealName{
		ID:       types.StringValue(data.ID.ValueString()),
		RealName: types.StringValue(slackUser.RealName),
	})...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Read Slack user real name", map[string]interface{}{
		"id":        data.ID.ValueString(),
		"real_name": slackUser.RealName,
	})
}

func (r *resourceSlackUserRealName) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_user_real_name** manage a Slack user real name.

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
			"real_name": schema.StringAttribute{
				MarkdownDescription: "The real name to set for the user.",
				Optional:            true,
			},
		},
	}
}

func (r *resourceSlackUserRealName) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserRealName

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.SetUserRealNameContextWithUser(ctx, data.ID.ValueString(), data.RealName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Slack User Real Name", err.Error())
		return
	}

	tflog.Trace(ctx, "Slack user real name updated", map[string]interface{}{
		"id":        data.ID.ValueString(),
		"real_name": data.RealName.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
