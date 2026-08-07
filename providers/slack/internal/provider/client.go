package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/slack-go/slack"
)

// ConfiguredClient holds the configuration for the provider, including the Slack API client.
type slackClient struct {
	*slack.Client
	RawResponse string //extended attribute
}

// Configure initializes the ConfiguredClient with the Slack API token.
func (c *slackClient) Configure(ctx context.Context, apiToken string, organization string) diag.Diagnostics {
	var diags diag.Diagnostics

	if apiToken == "" {
		diags.AddError("Missing API Token", "The SLACK_API_TOKEN environment variable or provider configuration is required.")
		return diags
	}

	// Initialize the Slack API client
	c.Client = slack.New(apiToken)

	// Add any additional logic for organization or other setup here

	return diags
}
