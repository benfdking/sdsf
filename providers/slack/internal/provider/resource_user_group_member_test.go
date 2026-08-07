package provider

import (
	"os"
	"testing"
)

func Test_resource_user_group_member(t *testing.T) {

	// Retrieve the token and user from env variables
	slackAPIToken := os.Getenv("SLACK_API_TOKEN")
	slackUserGroup := os.Getenv("SLACK_USER_GROUP")
	slackDefaultUser := os.Getenv("SLACK_DEFAULT_USER")
	slackUsers := os.Getenv("SLACK_USERS")

	if slackAPIToken == "" {
		t.Skip("SLACK_API_TOKEN environment variable not set, skipping test.")
	}

	if slackUserGroup == "" {
		t.Skip("SLACK_USER_GROUP environment variable not set, skipping test.")
	}

	if slackDefaultUser == "" {
		t.Skip("SLACK_DEFAULT_USER environment variable not set, skipping test.")
	}

	if slackUsers == "" {
		t.Skip("SLACK_USERS environment variable not set, skipping test.")
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

	// 				variable "slack_usergroup" {
	// 					type        = string
	// 					description = "The Slack user group name"
	// 					default     = "%s"
	// 				}

	// 				variable "slack_default_user" {
	// 					type        = string
	// 					description = "The default user email assigned to the specified Slack user group"
	// 					default     = "%s"
	// 				}

	// 				variable "slack_users" {
	// 					type        = string
	// 					description = "A list of users email to assign to the specified Slack user group"
	// 					default     = "%s"
	// 				}

	// 				locals {
	//   			  slack_users_list = split(",", var.slack_users)
	// 				}

	// 				resource "slack_user_group_member" "test" {
	// 					usergroup    = var.slack_usergroup
	// 					default_user = var.slack_default_user
	// 					users        = local.slack_users_list
	// 				}
	// 			`, slackAPIToken, slackUserGroup, slackDefaultUser, slackUsers),
	// 			ConfigPlanChecks: resource.ConfigPlanChecks{
	// 				PreApply: []plancheck.PlanCheck{},
	// 			},
	// 			Check: resource.TestCheckFunc(func(s *terraform.State) error {
	// 				// Retrieve the resource state
	// 				rs, ok := s.RootModule().Resources["slack_user_group_member.test"]
	// 				if !ok {
	// 					return fmt.Errorf("resource not found: slack_user_group_member.test")
	// 				}

	// 				// Check the properties of the resource
	// 				if rs.Primary.Attributes["usergroup"] != slackUserGroup {
	// 					return fmt.Errorf("expected id to be %s, got %s", slackUserGroup, rs.Primary.Attributes["usergroup"])
	// 				}
	// 				if rs.Primary.Attributes["default_user"] != slackDefaultUser {
	// 					return fmt.Errorf("expected default_user to be %s, got %s", slackDefaultUser, rs.Primary.Attributes["default_user"])
	// 				}
	// 				return nil
	// 			}),
	// 		},
	// 	},
	// })
}
