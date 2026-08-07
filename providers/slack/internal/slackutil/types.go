package slackutil

import (
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// StringValue represents a single string value.
type StringValue struct {
	Content string
}

// ListValue represents a list of StringValue objects and indicates whether the list is null.
type ListValue struct {
	Null     bool          // Indicates if the list is null.
	Elements []StringValue // The elements of the list.
}

// StringListType defines a custom type representing a list of string elements.
type StringListType struct {
	Elements []basetypes.StringValue
}

type MixedValue struct {
	StringContent string
	BoolContent   bool
	NumberContent *big.Float
}

// Users represents a list current state user information.
type Users struct {
	Emails []string
	IDs    []string
}

// Conversations represents a list current state conversation information.
type Conversations struct {
	Names []string
	IDs   []string
}

// ConversationDetails represents the details of a Slack conversation.
type ConversationDetails struct {
	Created            int64
	Creator            string
	ID                 string
	IsArchived         bool
	IsChannel          bool
	IsExtShared        bool
	IsGeneral          bool
	IsGroup            bool
	IsIM               bool
	IsMember           bool
	IsMpim             bool
	IsOrgShared        bool
	IsPendingExtShared bool
	IsPrivate          bool
	IsShared           bool
	Name               string
	NameNormalized     string
	NumMembers         int64
	Purpose            Purpose
	Topic              Topic
	Unlinked           int64
}

// Purpose represents the purpose of a Slack conversation.
type Purpose struct {
	Value   string
	Creator string
	LastSet int64
}

// Topic represents the topic of a Slack conversation.
type Topic struct {
	Value   string
	Creator string
	LastSet int64
}

// TeamInfo represents the details of a Slack team.
type TeamInfo struct {
	ID          string
	Name        string
	Domain      string
	EmailDomain string
	Icon        TeamIcon
}

// TeamIcon represents the icon images of a Slack team.
type TeamIcon struct {
	Image34      string
	ImageDefault bool
}
