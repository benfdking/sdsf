package provider

import (
	"os"
	"testing"
)

func Test_resource_slack_user_real_name(t *testing.T) {

	// Retrieve the token and user from env variables
	slackAPIToken := os.Getenv("SLACK_API_TOKEN")
	slackUserID := os.Getenv("SLACK_USER_ID")

	if slackAPIToken == "" {
		t.Skip("SLACK_API_TOKEN environment variable not set, skipping test.")
	}

	if slackUserID == "" {
		t.Skip("SLACK_USER_ID environment variable not set, skipping test.")
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

	// 				variable "slack_user_id" {
	// 					type        = string
	// 					description = "The Slack user ID"
	// 					default     = "%s"
	// 				}

	// 				resource "slack_user_real_name" "test" {
	// 					id        = var.slack_user_id
	// 					real_name = "testuser01"
	// 				}
	// 			`, slackAPIToken, slackUserID),
	// 			ConfigPlanChecks: resource.ConfigPlanChecks{
	// 				PreApply: []plancheck.PlanCheck{},
	// 			},
	// 			Check: resource.TestCheckFunc(func(s *terraform.State) error {
	// 				// Retrieve the resource state
	// 				rs, ok := s.RootModule().Resources["slack_user_real_name.test"]
	// 				if !ok {
	// 					return fmt.Errorf("resource not found: slack_user_real_name.test")
	// 				}

	// 				// Check the properties of the resource
	// 				if rs.Primary.Attributes["id"] != slackUserID {
	// 					return fmt.Errorf("expected id to be %s, got %s", slackUserID, rs.Primary.Attributes["id"])
	// 				}
	// 				if rs.Primary.Attributes["real_name"] != "testuser01" {
	// 					return fmt.Errorf("expected real_name to be 'testuser01', got %s", rs.Primary.Attributes["real_name"])
	// 				}

	// 				return nil
	// 			}),
	// 		},
	// 	},
	// })
}
