package slackutil

import (
	"fmt"

	"github.com/slack-go/slack"
)

func GetConversationNames(api *slack.Client, channelIds []string, channelTypes []string, limit int) ([]string, error) {

	var channelNames []string
	for _, channelId := range channelIds {
		conversation, err := GetConversation(api, channelId, "id", true, channelTypes, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve conversation names for channel '%s': %v", channelId, err)
		}
		channelNames = append(channelNames, conversation.Name)
	}

	return channelNames, nil
}
