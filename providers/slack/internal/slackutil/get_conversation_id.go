package slackutil

import (
	"fmt"

	"github.com/slack-go/slack"
)

func GetConversationIds(api *slack.Client, channelNames []string, channelTypes []string, limit int) ([]string, error) {

	var channelIds []string
	for _, channelName := range channelNames {
		conversation, err := GetConversation(api, channelName, "name", true, channelTypes, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve conversation id for channel '%s': %v", channelName, err)
		}
		channelIds = append(channelIds, conversation.ID)
	}

	return channelIds, nil
}
