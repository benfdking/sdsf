package provider

import (
	"os"
	"testing"
)

func Test_resource_user_group(t *testing.T) {

	// Retrieve the token and user from env variables
	slackAPIToken := os.Getenv("SLACK_API_TOKEN")

	if slackAPIToken == "" {
		t.Skip("SLACK_API_TOKEN environment variable not set, skipping test.")
	}

	// commented as this creates real resource

	// resource.UnitTest(t, resource.TestCase{
	// 	TerraformVersionChecks: []tfversion.TerraformVersionCheck{
	// 		tfversion.SkipBelow(tfversion.Version1_8_0),
	// 	},
	// 	ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
	// 	Steps: []resource.TestStep{
	// 		{
	// 			Config: fmt.Sprintf(`
	// 				terraform {
	// 					required_providers {
	// 						slack = {
	// 							source = "benfdking/slack"
	// 						}
	// 					}
	// 				}

	// 				provider "slack" {
	// 					api_token = var.slack_api_token
	// 				}

	// 				variable "slack_api_token" {
	// 					type        = string
	// 					description = "The API token for authenticating with Slack"
	// 					default     = "%s"
	// 				}

	// 				resource "slack_user_group" "test" {
	// 					name    = "Test Group Z"
	// 					description = "test group Z"
	// 					handle      = "test-team-z"
	// 					channels    = ["open"]
	// 				}
	// 			`, slackAPIToken),
	// 			ConfigPlanChecks: resource.ConfigPlanChecks{
	// 				PreApply: []plancheck.PlanCheck{},
	// 			},
	// 			Check: resource.TestCheckFunc(func(s *terraform.State) error {
	// 				// Retrieve the resource state
	// 				_, ok := s.RootModule().Resources["slack_user_group.test"]
	// 				if !ok {
	// 					return fmt.Errorf("resource not found: slack_user_group.test")
	// 				}
	// 				return nil
	// 			}),
	// 		},
	// 	},
	// })
}
