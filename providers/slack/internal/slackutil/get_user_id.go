package slackutil

import (
	"fmt"

	"github.com/slack-go/slack"
)

// GetUserIds retrieves a list of Slack user IDs based on a list of email addresses.
//
// This function takes a Slack API client and a slice of email addresses and uses the
// provided client to find users associated with each email address. For each email,
// it uses GetUserAttributes to look up the user, then appends the user's ID to the result list.
// If an error occurs when retrieving a user, it returns an error indicating which email lookup failed.
//
// Parameters:
//   - api: A pointer to the slack.Client used to interact with the Slack API.
//   - emails: A slice of email addresses for which to retrieve Slack user IDs.
//
// Returns:
//   - A pointer to a Users struct containing the list of emails and corresponding user IDs.
//   - An error if there was an issue retrieving any user ID.
//
// Example usage:
//
//	api := slack.New("YOUR_SLACK_BOT_TOKEN")
//	emails := []string{"user1@example.com", "user2@example.com"}
//	userIDs, err := GetUserIds(api, emails)
//	if err != nil {
//	    log.Fatalf("Error: %v", err)
//	}
//	fmt.Printf("Emails: %v, IDs: %v\n", userIDs.Emails, userIDs.IDs)
func GetUserIds(api *slack.Client, emails []string) (*Users, error) {
	var userIds []string
	for _, email := range emails {
		user, err := GetUserAttributes(api, "email", email)
		if err != nil {
			return nil, fmt.Errorf("failed to get user details for email %s: %w", email, err)
		}
		userIds = append(userIds, user.ID)
	}

	uga := &Users{
		Emails: emails,
		IDs:    userIds,
	}

	return uga, nil
}
