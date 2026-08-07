package slackutil

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

// SlackUserGroupAttributes defines the attributes for a Slack user group.
//
// This struct holds the necessary attributes for a Slack user group,
// which can be returned from the GetUserGroupByName function.
type SlackUserGroupAttributes struct {
	AutoType    string         `tfsdk:"auto_type"`
	Channels    []types.String `tfsdk:"channels"`
	CreatedBy   string         `tfsdk:"created_by"`
	DateCreate  int64          `tfsdk:"date_create"`
	DateDelete  int64          `tfsdk:"date_delete"`
	DateUpdate  int64          `tfsdk:"date_update"`
	DeletedBy   string         `tfsdk:"deleted_by"`
	Description string         `tfsdk:"description"`
	Groups      []types.String `tfsdk:"groups"`
	Handle      string         `tfsdk:"handle"`
	ID          string         `tfsdk:"id"`
	IsExternal  bool           `tfsdk:"is_external"`
	IsUserGroup bool           `tfsdk:"is_user_group"`
	Name        string         `tfsdk:"name"`
	TeamID      string         `tfsdk:"team_id"`
	UpdatedBy   string         `tfsdk:"updated_by"`
	UserCount   int64          `tfsdk:"user_count"`
	Users       []types.String `tfsdk:"users"`
}

// GetUserGroupByName retrieves a Slack user group by its name and returns its attributes.
//
// This function queries the Slack API for user groups, looking for a group that matches the provided name.
// If a matching group is found, its attributes are returned along with a boolean indicating its existence.
// If no group is found, the function returns a SlackUserGroupAttributes struct with default (empty) values,
// while still indicating that the group does not exist.
//
// Sample Input:
//
//	filterUserGroupName := types.StringValue("example_group")
//
// Sample Output:
//
//	groupAttributes, exists, err := GetUserGroupByName(client, filterUserGroupName)
//	// groupAttributes will contain the attributes of the found group or default values if not found.
//
// Returns:
//
//	A pointer to SlackUserGroupAttributes containing the details of the user group if found,
//	or default attributes if not found. The boolean indicates whether the group exists.
//	If an error occurs while retrieving the user groups, an error is returned.
func GetUserGroupByName(client *slack.Client, filterUserGroupName types.String) (*SlackUserGroupAttributes, bool, error) {
	// Get user groups from Slack
	userGroups, err := client.GetUserGroups(
		slack.GetUserGroupsOptionIncludeUsers(true),
		slack.GetUserGroupsOptionIncludeDisabled(true),
	)
	if err != nil {
		errorMsg := "An error occurred while retrieving the user groups: " + err.Error()
		fmt.Printf("API error: %s\n", errorMsg)
		return nil, false, err
	}

	// Check if a matching group is found
	var group *slack.UserGroup
	for _, g := range userGroups {
		if !filterUserGroupName.IsNull() && g.Name == filterUserGroupName.ValueString() {
			gCopy := g
			group = &gCopy
			break
		}
	}

	// If no matching group is found, return default empty attributes and false
	if group == nil {
		defaultGroup := &SlackUserGroupAttributes{} // Create default attributes
		return defaultGroup, false, nil
	}

	// Populate the attributes from the found user group
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

	// Return the populated user group attributes along with existence status
	return &SlackUserGroupAttributes{
		AutoType:    group.AutoType,
		Channels:    channels,
		CreatedBy:   group.CreatedBy,
		DateCreate:  int64(group.DateCreate),
		DateDelete:  int64(group.DateDelete),
		DateUpdate:  int64(group.DateUpdate),
		DeletedBy:   group.DeletedBy,
		Description: group.Description,
		Groups:      groups,
		Handle:      group.Handle,
		ID:          group.ID,
		IsExternal:  group.IsExternal,
		IsUserGroup: group.IsUserGroup,
		Name:        group.Name,
		TeamID:      group.TeamID,
		UpdatedBy:   group.UpdatedBy,
		UserCount:   int64(group.UserCount),
		Users:       users,
	}, true, nil
}
