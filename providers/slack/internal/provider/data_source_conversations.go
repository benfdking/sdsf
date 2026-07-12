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

type dataSourceConversations struct {
	client *slack.Client
}

func NewdataSourceConversations() datasource.DataSource {
	return &dataSourceConversations{}
}

func (d *dataSourceConversations) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		providerClient, ok := req.ProviderData.(*slackClient)
		if !ok {
			resp.Diagnostics.AddError("Invalid Provider Data", "Expected *ConfiguredClient but got something else.")
			return
		}
		d.client = providerClient.Client
	}
}

func (d *dataSourceConversations) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "slack_conversations"
}

func (d *dataSourceConversations) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state struct {
		ExcludeArchived types.Bool  `tfsdk:"exclude_archived"`
		Types           types.List  `tfsdk:"types"`
		QueryLimit      types.Int64 `tfsdk:"query_limit"`
		Conversations   []struct {
			Created            types.Int64  `tfsdk:"created"`
			Creator            types.String `tfsdk:"creator"`
			ID                 types.String `tfsdk:"id"`
			IsArchived         types.Bool   `tfsdk:"is_archived"`
			IsChannel          types.Bool   `tfsdk:"is_channel"`
			IsExtShared        types.Bool   `tfsdk:"is_ext_shared"`
			IsGeneral          types.Bool   `tfsdk:"is_general"`
			IsGroup            types.Bool   `tfsdk:"is_group"`
			IsIM               types.Bool   `tfsdk:"is_im"`
			IsMember           types.Bool   `tfsdk:"is_member"`
			IsMpim             types.Bool   `tfsdk:"is_mpim"`
			IsOrgShared        types.Bool   `tfsdk:"is_org_shared"`
			IsPendingExtShared types.Bool   `tfsdk:"is_pending_ext_shared"`
			IsPrivate          types.Bool   `tfsdk:"is_private"`
			IsShared           types.Bool   `tfsdk:"is_shared"`
			Name               types.String `tfsdk:"name"`
			NameNormalized     types.String `tfsdk:"name_normalized"`
			NumMembers         types.Int64  `tfsdk:"num_members"`
			Purpose            Purpose      `tfsdk:"purpose"`
			Topic              Topic        `tfsdk:"topic"`
			Unlinked           types.Int64  `tfsdk:"unlinked"`
		} `tfsdk:"conversations"`
	}

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conversationTypes := []string{"public_channel"}
	validConversationTypes := []string{"public_channel", "private_channel", "mpim", "im"}
	if !state.Types.IsUnknown() && len(state.Types.Elements()) > 0 {
		// convert []attrValue to []string
		types, err := slackutil.ConvertAttrValuesToStrings(state.Types.Elements())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error converting users email",
				fmt.Sprintf("An error occurred while converting users email to Strings: %s", err.Error()),
			)
			return
		}

		// merge and remove duplicate
		conversationTypes, err = slackutil.MergeAndValidateStrings(conversationTypes, types)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error merging and validating conversation types",
				fmt.Sprintf("An error occurred while merging conversation types: %s", err.Error()),
			)
			return
		}

		// comparing types containing only valid
		cmp, err := slackutil.CompareStrings(validConversationTypes, conversationTypes, slackutil.Subset)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error comparing conversation types",
				fmt.Sprintf("An error occurred while comparing conversation types: %s", err.Error()),
			)
			return
		}

		if !cmp {
			resp.Diagnostics.AddError(
				"Invalid conversation types",
				fmt.Sprintf("Allowed conversation types are: %v", validConversationTypes),
			)
			return
		}
	}

	limit := 1000
	if !state.QueryLimit.IsNull() {
		queryLimitValue := state.QueryLimit.ValueInt64()
		limit = int(queryLimitValue)
	}

	params := &slack.GetConversationsParameters{
		ExcludeArchived: state.ExcludeArchived.ValueBool(),
		Types:           conversationTypes,
		Limit:           limit,
	}

	var allConversations []slack.Channel
	for {
		conversations, nextCursor, err := d.client.GetConversations(params)
		if err != nil {
			resp.Diagnostics.AddError("Error fetching conversations", err.Error())
			return
		}
		allConversations = append(allConversations, conversations...)

		if nextCursor == "" {
			break
		}
		params.Cursor = nextCursor
	}

	state.Conversations = make([]struct {
		Created            types.Int64  `tfsdk:"created"`
		Creator            types.String `tfsdk:"creator"`
		ID                 types.String `tfsdk:"id"`
		IsArchived         types.Bool   `tfsdk:"is_archived"`
		IsChannel          types.Bool   `tfsdk:"is_channel"`
		IsExtShared        types.Bool   `tfsdk:"is_ext_shared"`
		IsGeneral          types.Bool   `tfsdk:"is_general"`
		IsGroup            types.Bool   `tfsdk:"is_group"`
		IsIM               types.Bool   `tfsdk:"is_im"`
		IsMember           types.Bool   `tfsdk:"is_member"`
		IsMpim             types.Bool   `tfsdk:"is_mpim"`
		IsOrgShared        types.Bool   `tfsdk:"is_org_shared"`
		IsPendingExtShared types.Bool   `tfsdk:"is_pending_ext_shared"`
		IsPrivate          types.Bool   `tfsdk:"is_private"`
		IsShared           types.Bool   `tfsdk:"is_shared"`
		Name               types.String `tfsdk:"name"`
		NameNormalized     types.String `tfsdk:"name_normalized"`
		NumMembers         types.Int64  `tfsdk:"num_members"`
		Purpose            Purpose      `tfsdk:"purpose"`
		Topic              Topic        `tfsdk:"topic"`
		Unlinked           types.Int64  `tfsdk:"unlinked"`
	}, len(allConversations))

	for i, conversation := range allConversations {
		state.Conversations[i] = struct {
			Created            types.Int64  `tfsdk:"created"`
			Creator            types.String `tfsdk:"creator"`
			ID                 types.String `tfsdk:"id"`
			IsArchived         types.Bool   `tfsdk:"is_archived"`
			IsChannel          types.Bool   `tfsdk:"is_channel"`
			IsExtShared        types.Bool   `tfsdk:"is_ext_shared"`
			IsGeneral          types.Bool   `tfsdk:"is_general"`
			IsGroup            types.Bool   `tfsdk:"is_group"`
			IsIM               types.Bool   `tfsdk:"is_im"`
			IsMember           types.Bool   `tfsdk:"is_member"`
			IsMpim             types.Bool   `tfsdk:"is_mpim"`
			IsOrgShared        types.Bool   `tfsdk:"is_org_shared"`
			IsPendingExtShared types.Bool   `tfsdk:"is_pending_ext_shared"`
			IsPrivate          types.Bool   `tfsdk:"is_private"`
			IsShared           types.Bool   `tfsdk:"is_shared"`
			Name               types.String `tfsdk:"name"`
			NameNormalized     types.String `tfsdk:"name_normalized"`
			NumMembers         types.Int64  `tfsdk:"num_members"`
			Purpose            Purpose      `tfsdk:"purpose"`
			Topic              Topic        `tfsdk:"topic"`
			Unlinked           types.Int64  `tfsdk:"unlinked"`
		}{
			Created:            types.Int64Value(int64(conversation.Created)),
			Creator:            types.StringValue(conversation.Creator),
			ID:                 types.StringValue(conversation.ID),
			IsArchived:         types.BoolValue(conversation.IsArchived),
			IsChannel:          types.BoolValue(conversation.IsChannel),
			IsExtShared:        types.BoolValue(conversation.IsExtShared),
			IsGeneral:          types.BoolValue(conversation.IsGeneral),
			IsGroup:            types.BoolValue(conversation.IsGroup),
			IsIM:               types.BoolValue(conversation.IsIM),
			IsMember:           types.BoolValue(conversation.IsMember),
			IsMpim:             types.BoolValue(conversation.IsMpIM),
			IsOrgShared:        types.BoolValue(conversation.IsOrgShared),
			IsPendingExtShared: types.BoolValue(conversation.IsPendingExtShared),
			IsPrivate:          types.BoolValue(conversation.IsPrivate),
			IsShared:           types.BoolValue(conversation.IsShared),
			Name:               types.StringValue(conversation.Name),
			NameNormalized:     types.StringValue(conversation.NameNormalized),
			NumMembers:         types.Int64Value(int64(conversation.NumMembers)),
			Purpose: Purpose{
				Value:   types.StringValue(defaultIfEmpty(conversation.Purpose.Value, "No Purpose")),
				Creator: types.StringValue(conversation.Purpose.Creator),
				LastSet: types.Int64Value(int64(conversation.Purpose.LastSet)),
			},
			Topic: Topic{
				Value:   types.StringValue(defaultIfEmpty(conversation.Topic.Value, "No Topic")),
				Creator: types.StringValue(conversation.Topic.Creator),
				LastSet: types.Int64Value(int64(conversation.Topic.LastSet)),
			},
			Unlinked: types.Int64Value(int64(conversation.Unlinked)),
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *dataSourceConversations) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The **slack_conversations** data source retrieves a list of conversations (channels) in a Slack workspace.

You can customize the data retrieval using optional parameters such as excluding archived conversations and filtering by conversation types.

**Required scopes**

User tokens: channels:read, groups:read, im:read, mpim:read
`,
		Attributes: map[string]schema.Attribute{
			"exclude_archived": schema.BoolAttribute{
				MarkdownDescription: "Exclude archived conversations from the list.",
				Optional:            true,
			},
			"query_limit": schema.Int64Attribute{
				MarkdownDescription: `
Maximum number of items to query.

This limit controls how many items are returned in a single query. Setting a higher limit may increase the response time, while a lower limit can help optimize performance and reduce resource usage.

**Optional:** If not specified, the default limit is 1000.
`,
				Optional: true,
			},
			"types": schema.ListAttribute{
				MarkdownDescription: `
Types of conversation to include (e.g., 'public_channel', 'private_channel')."

Default: 'public_channel'
`,
				ElementType: types.StringType,
				Optional:    true,
			},
			"conversations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of Slack conversations.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"created": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Timestamp of when the conversation was created.",
						},
						"creator": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User ID of the creator of the conversation.",
						},
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique identifier of the conversation (channel).",
						},
						"is_archived": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is archived.",
						},
						"is_channel": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is a Slack channel.",
						},
						"is_ext_shared": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is shared externally.",
						},
						"is_general": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if this is the general conversation.",
						},
						"is_group": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is a group (private).",
						},
						"is_im": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is a direct message.",
						},
						"is_member": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the authenticated user is a member of the conversation.",
						},
						"is_mpim": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is a multi-party instant message (group DM).",
						},
						"is_org_shared": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is shared across an organization.",
						},
						"is_pending_ext_shared": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is pending external sharing.",
						},
						"is_private": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is private.",
						},
						"is_shared": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "True if the conversation is shared.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the conversation (channel).",
						},
						"name_normalized": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The normalized name of the conversation.",
						},
						"num_members": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of members in the conversation.",
						},
						"purpose": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									Computed:    true,
									Description: "The purpose of the conversation.",
								},
								"creator": schema.StringAttribute{
									Computed:    true,
									Description: "The user who set the purpose.",
								},
								"last_set": schema.Int64Attribute{
									Computed:    true,
									Description: "Timestamp of when the purpose was last set.",
								},
							},
						},
						"topic": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									Computed:    true,
									Description: "The topic of the conversation.",
								},
								"creator": schema.StringAttribute{
									Computed:    true,
									Description: "The user who set the topic.",
								},
								"last_set": schema.Int64Attribute{
									Computed:    true,
									Description: "Timestamp of when the topic was last set.",
								},
							},
						},
						"unlinked": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The number of messages unlinked from external sources.",
						},
					},
				},
			},
		},
	}
}
