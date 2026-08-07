package slackutil

import (
	"fmt"

	"github.com/slack-go/slack"
)

// GetConversation retrieves the details of a Slack conversation by its name or ID.
// The filterType specifies whether to use "name" or "id" for filtering.
// If "name" is provided, it will search for the conversation by name. If "id" is provided,
// it will search for the conversation by ID.
//
// Parameters:
//   - api: A pointer to the slack.Client used to interact with the Slack API.
//   - filter: The name or ID of the conversation to search for. If both are provided, name takes precedence.
//   - filterType: A string indicating whether to filter by "name" or "id".
//   - excludeArchived: A boolean indicating whether to exclude archived conversations.
//   - types: A slice of strings representing the types of conversations to include (e.g., channels, groups).
//   - queryLimit: An integer specifying the maximum number of conversations to fetch.
//
// Returns:
//   - A pointer to ConversationDetails if the conversation is found.
//   - An error if there was an issue retrieving the conversations or if the conversation is not found.
func GetConversation(api *slack.Client, filter string, filterType string, excludeArchived bool, types []string, queryLimit int) (*ConversationDetails, error) {
	var foundConversation *slack.Channel

	// Set a default limit for the number of conversations to fetch
	limit := 1000
	if queryLimit > 0 {
		limit = queryLimit
	}

	params := &slack.GetConversationsParameters{
		ExcludeArchived: excludeArchived,
		Types:           types,
		Limit:           limit,
	}

	var allConversations []slack.Channel
	for {
		conversations, nextCursor, err := api.GetConversations(params)
		if err != nil {
			return nil, fmt.Errorf("error fetching conversations: %w", err)
		}
		allConversations = append(allConversations, conversations...)

		if nextCursor == "" {
			break
		}
		params.Cursor = nextCursor
	}

	// Filter conversations by ID or name based on filterType
	for _, conversation := range allConversations {
		if filterType == "name" && conversation.Name == filter {
			convCopy := conversation
			foundConversation = &convCopy
			break
		} else if filterType == "id" && conversation.ID == filter {
			convCopy := conversation
			foundConversation = &convCopy
			break
		}
	}

	if foundConversation == nil {
		return nil, fmt.Errorf("no conversation found matching the provided %s '%s'",
			filterType, filter)
	}

	// Populate the conversation details
	return &ConversationDetails{
		Created:            int64(foundConversation.Created),
		Creator:            foundConversation.Creator,
		ID:                 foundConversation.ID,
		IsArchived:         foundConversation.IsArchived,
		IsChannel:          foundConversation.IsChannel,
		IsExtShared:        foundConversation.IsExtShared,
		IsGeneral:          foundConversation.IsGeneral,
		IsGroup:            foundConversation.IsGroup,
		IsIM:               foundConversation.IsIM,
		IsMember:           foundConversation.IsMember,
		IsMpim:             foundConversation.IsMpIM,
		IsOrgShared:        foundConversation.IsOrgShared,
		IsPendingExtShared: foundConversation.IsPendingExtShared,
		IsPrivate:          foundConversation.IsPrivate,
		IsShared:           foundConversation.IsShared,
		Name:               foundConversation.Name,
		NameNormalized:     foundConversation.NameNormalized,
		NumMembers:         int64(foundConversation.NumMembers),
		Purpose: Purpose{
			Value:   defaultIfEmpty(foundConversation.Purpose.Value, "No Purpose"),
			Creator: foundConversation.Purpose.Creator,
			LastSet: int64(foundConversation.Purpose.LastSet),
		},
		Topic: Topic{
			Value:   defaultIfEmpty(foundConversation.Topic.Value, "No Topic"),
			Creator: foundConversation.Topic.Creator,
			LastSet: int64(foundConversation.Topic.LastSet),
		},
		Unlinked: int64(foundConversation.Unlinked),
	}, nil
}

// defaultIfEmpty returns the default value if the given value is empty.
func defaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
