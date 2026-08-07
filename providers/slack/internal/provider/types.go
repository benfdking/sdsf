package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

// User - start

type EnterpriseUser struct {
	EnterpriseID   types.String   `tfsdk:"enterprise_id"`
	EnterpriseName types.String   `tfsdk:"enterprise_name"`
	ID             types.String   `tfsdk:"id"`
	IsAdmin        types.Bool     `tfsdk:"is_admin"`
	IsOwner        types.Bool     `tfsdk:"is_owner"`
	Teams          []types.String `tfsdk:"teams"`
}

type UserProfile struct {
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
}

type User struct {
	Color             types.String   `tfsdk:"color"`
	Deleted           types.Bool     `tfsdk:"deleted"`
	Enterprise        EnterpriseUser `tfsdk:"enterprise_user"`
	Has2FA            types.Bool     `tfsdk:"has_2fa"`
	HasFiles          types.Bool     `tfsdk:"has_files"`
	ID                types.String   `tfsdk:"id"`
	IsAdmin           types.Bool     `tfsdk:"is_admin"`
	IsAppUser         types.Bool     `tfsdk:"is_app_user"`
	IsBot             types.Bool     `tfsdk:"is_bot"`
	IsInvitedUser     types.Bool     `tfsdk:"is_invited_user"`
	IsOwner           types.Bool     `tfsdk:"is_owner"`
	IsPrimaryOwner    types.Bool     `tfsdk:"is_primary_owner"`
	IsRestricted      types.Bool     `tfsdk:"is_restricted"`
	IsStranger        types.Bool     `tfsdk:"is_stranger"`
	IsUltraRestricted types.Bool     `tfsdk:"is_ultra_restricted"`
	Locale            types.String   `tfsdk:"locale"`
	Name              types.String   `tfsdk:"name"`
	Presence          types.String   `tfsdk:"presence"`
	Profile           UserProfile    `tfsdk:"profile"`
	RealName          types.String   `tfsdk:"real_name"`
	TeamID            types.String   `tfsdk:"team_id"`
	TZ                types.String   `tfsdk:"tz"`
	TZLabel           types.String   `tfsdk:"tz_label"`
	TZOffset          types.Int64    `tfsdk:"tz_offset"`
	Updated           slack.JSONTime `tfsdk:"updated"`
}

// User - end

// Usergroup - start

type GetUserGroupsOption struct {
	IncludeUsers    bool   `json:"include_users_filter"`
	IncludeCount    bool   `json:"include_count_filter"`
	IncludeDisabled bool   `json:"include_disabled_filter"`
	TeamID          string `json:"team_id_filter"`
}

// Usergroup - end

// Conversation - start

type Purpose struct {
	Value   types.String `tfsdk:"value"`
	Creator types.String `tfsdk:"creator"`
	LastSet types.Int64  `tfsdk:"last_set"`
}

type Topic struct {
	Value   types.String `tfsdk:"value"`
	Creator types.String `tfsdk:"creator"`
	LastSet types.Int64  `tfsdk:"last_set"`
}

type Conversation struct {
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
}

// Conversations - end
