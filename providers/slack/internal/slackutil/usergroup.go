package slackutil

import (
	"fmt"

	"github.com/slack-go/slack"
)

type UserGroupAttributes struct {
	AutoType    string
	Channels    []string
	CreatedBy   string
	DateCreate  int64
	DateDelete  int64
	DateUpdate  int64
	DeletedBy   string
	Description string
	Groups      []string
	Handle      string
	ID          string
	IsExternal  bool
	IsUsergroup bool
	Name        string
	TeamID      string
	UpdatedBy   string
	UserCount   int
	UserIds     []string
	UserEmails  []string
}

// GetUserGroupAttributes retrieves the attributes of a Slack user group by its name.
// This function uses the provided Slack API client to fetch all user groups and searches
// for the group that matches the specified name. If the group is found, it returns a
// pointer to a UserGroupAttributes struct containing the group's attributes, including
// the ID, name, description, handle, and auto type. If the group is not found or an error
// occurs while fetching the user groups, it returns an error.
//
// Parameters:
//   - api: A pointer to the slack.Client used to interact with the Slack API.
//   - groupName: The name of the user group to search for.
//
// Returns:
//   - A pointer to UserGroupAttributes if the user group is found.
//   - An error if there was an issue retrieving the user groups or if the group is not found.
//
// Example usage:
//
//	api := slack.New("YOUR_SLACK_BOT_TOKEN")
//	groupName := "desired_user_group_name"
//	groupAttributes, err := GetUserGroupAttributes(api, groupName)
//	if err != nil {
//	    log.Fatalf("Error: %v", err)
//	}
//	fmt.Printf("User Group ID: %s\n", groupAttributes.ID)
func GetUserGroupAttributes(api *slack.Client, groupName string) (*UserGroupAttributes, error) {
	// Fetch the list of user groups
	userGroups, err := api.GetUserGroups(
		slack.GetUserGroupsOptionIncludeUsers(true),
		slack.GetUserGroupsOptionIncludeCount(true),
		slack.GetUserGroupsOptionIncludeDisabled(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	// Iterate through the user groups to find the one with the given name
	for _, group := range userGroups {
		if group.Name == groupName {
			// Create a UserGroupAttributes instance
			uga := &UserGroupAttributes{
				AutoType:    group.AutoType,
				Channels:    group.Prefs.Channels,
				CreatedBy:   group.CreatedBy,
				DateCreate:  int64(group.DateCreate),
				DateDelete:  int64(group.DateDelete),
				DateUpdate:  int64(group.DateUpdate),
				DeletedBy:   group.DeletedBy,
				Description: group.Description,
				Groups:      group.Prefs.Groups,
				Handle:      group.Handle,
				ID:          group.ID,
				IsExternal:  group.IsExternal,
				IsUsergroup: group.IsUserGroup,
				Name:        group.Name,
				TeamID:      group.TeamID,
				UpdatedBy:   group.UpdatedBy,
				UserCount:   group.UserCount,
				UserIds:     group.Users,
			}

			// Call GetUserEmails to populate UserEmails
			if _, err := uga.GetUserEmails(api); err != nil {
				return nil, fmt.Errorf("failed to get emails for user group '%s': %w", groupName, err)
			}

			// Return the populated group attributes
			return uga, nil
		}
	}

	return nil, fmt.Errorf("user group '%s' not found", groupName)
}

// GetUserEmails retrieves the email addresses of users associated with the user group.
// It uses the UserIds to fetch the user details from the Slack API.
//
// Parameters:
// - api: A pointer to a slack.Client for making API calls.
//
// Returns:
// - A slice of strings containing email addresses of users in the group.
// - An error if any occurred during the process of retrieving user details.
func (uga *UserGroupAttributes) GetUserEmails(api *slack.Client) ([]string, error) {
	var emails []string
	for _, userId := range uga.UserIds { // Use UserIds from the populated UserGroupAttributes
		user, err := GetUserAttributes(api, "id", userId)
		if err != nil {
			return nil, fmt.Errorf("failed to get user details for ID %s: %w", userId, err)
		}
		emails = append(emails, user.Email)
	}

	uga.UserEmails = emails
	return emails, nil
}
