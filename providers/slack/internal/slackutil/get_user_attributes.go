package slackutil

import (
	"fmt"

	"github.com/slack-go/slack"
)

// UserAttributes represents the attributes of a Slack user.
type UserAttributes struct {
	ID       string
	Name     string
	Email    string
	RealName string
	IsBot    bool
}

// GetUserAttributes retrieves the attributes of a Slack user by either their email or ID.
//
// This function uses the provided Slack API client to fetch all users and searches
// for the user that matches the specified email or ID based on the filter type. If the user is found,
// it returns a pointer to a UserAttributes struct containing the user's attributes, including
// the ID, name, email, real name, and whether the user is a bot. If the user is
// not found or an error occurs while fetching the user list, it returns an error.
//
// Parameters:
//   - api: A pointer to the slack.Client used to interact with the Slack API.
//   - value: The email address or user ID of the user to search for.
//   - filterType: The type of filter to apply, either "email" or "id".
//
// Returns:
//   - A pointer to UserAttributes if the user is found.
//   - An error if there was an issue retrieving the users or if the user is not found.
//
// Example usage:
//
//	api := slack.New("YOUR_SLACK_BOT_TOKEN")
//	email := "user@example.com"
//	userAttributes, err := GetUserAttributes(api, email, "email")
//	if err != nil {
//	    log.Fatalf("Error: %v", err)
//	}
//	fmt.Printf("User ID: %s, Name: %s\n", userAttributes.ID, userAttributes.Name)
//
//	api := slack.New("YOUR_SLACK_BOT_TOKEN")
//	id := "U12345"
//	userAttributes, err := GetUserAttributes(api, id, "id")
//	if err != nil {
//	    log.Fatalf("Error: %v", err)
//	}
//	fmt.Printf("User ID: %s, Name: %s\n", userAttributes.ID, userAttributes.Name)
func GetUserAttributes(api *slack.Client, filterType string, value string) (*UserAttributes, error) {
	// Fetch the list of users
	users, err := api.GetUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Iterate through the users to find the one based on the filter type
	for _, user := range users {
		switch filterType {
		case "email":
			if user.Profile.Email == value {
				// Return the user attributes
				return &UserAttributes{
					ID:       user.ID,
					Name:     user.Name,
					Email:    user.Profile.Email,
					RealName: user.Profile.RealName,
					IsBot:    user.IsBot,
				}, nil
			}
		case "id":
			if user.ID == value {
				// Return the user attributes
				return &UserAttributes{
					ID:       user.ID,
					Name:     user.Name,
					Email:    user.Profile.Email,
					RealName: user.Profile.RealName,
					IsBot:    user.IsBot,
				}, nil
			}
		default:
			return nil, fmt.Errorf("invalid filter type '%s', expected 'email' or 'id'", filterType)
		}
	}

	// If no user was found
	return nil, fmt.Errorf("user with %s '%s' not found", filterType, value)
}
