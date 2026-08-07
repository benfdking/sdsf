package slackutil

import (
	"fmt"

	"github.com/slack-go/slack"
)

// GetUserEmails retrieves the email addresses associated with a list of Slack user IDs.
//
// This function uses the provided Slack API client to retrieve the email addresses
// of users based on their user IDs. For each ID in the input slice, it calls
// GetUserAttributes to fetch the user details and extract the email. It then
// constructs a Users struct containing the list of requested IDs and the corresponding emails.
// If any user is not found or an error occurs while fetching user details, it returns an error.
//
// Parameters:
//   - api: A pointer to the slack.Client used to interact with the Slack API.
//   - ids: A slice of strings containing the user IDs for which to retrieve emails.
//
// Returns:
//   - A pointer to a Users struct containing the Emails and IDs fields populated.
//   - An error if there was an issue retrieving any user or if a user is not found.
//
// Example usage:
//
//	api := slack.New("YOUR_SLACK_BOT_TOKEN")
//	ids := []string{"U12345", "U67890"}
//	userEmails, err := GetUserEmails(api, ids)
//	if err != nil {
//	    log.Fatalf("Error: %v", err)
//	}
//	fmt.Printf("User Emails: %v\n", userEmails.Emails)
func GetUserEmails(api *slack.Client, ids []string) (*Users, error) {
	var userEmails []string
	for _, id := range ids {
		user, err := GetUserAttributes(api, "id", id)
		if err != nil {
			return nil, fmt.Errorf("failed to get user details for ID %s: %w", id, err)
		}
		userEmails = append(userEmails, user.Email)
	}

	uga := &Users{
		Emails: userEmails,
		IDs:    ids,
	}

	return uga, nil
}
