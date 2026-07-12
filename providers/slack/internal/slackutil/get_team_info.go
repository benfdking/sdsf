package slackutil

import (
	"fmt"

	"github.com/slack-go/slack"
)

// GetTeamInfo retrieves the details of a Slack team.
//
// Parameters:
//   - api: A pointer to the slack.Client used to interact with the Slack API.
//
// Returns:
//   - A pointer to TeamInfo containing details about the team.
//   - An error if there was an issue retrieving the team info.
//
// Example usage:
//
//	api := slack.New("YOUR_SLACK_BOT_TOKEN")
//	teamInfo, err := slackutil.GetTeamInfo(api)
//	if err != nil {
//	    log.Fatalf("Error: %v", err)
//	}
//	fmt.Printf("Team ID: %s, Name: %s, Domain: %s\n", teamInfo.ID, teamInfo.Name, teamInfo.Domain)
//	fmt.Printf("Team Icon (132px): %s\n", teamInfo.Icon.Image132)
func GetTeamInfo(api *slack.Client) (*TeamInfo, error) {
	team, err := api.GetTeamInfo()
	if err != nil {
		return nil, fmt.Errorf("error fetching team info: %w", err)
	}

	// Extract icon URLs from the map
	icon := TeamIcon{
		Image34:      GetStringFromMap(team.Icon, "image_34"),
		ImageDefault: GetBoolFromMap(team.Icon, "image_default"),
	}

	// Populate and return the team information
	return &TeamInfo{
		ID:          team.ID,
		Name:        team.Name,
		Domain:      team.Domain,
		EmailDomain: team.EmailDomain,
		Icon:        icon,
	}, nil
}
