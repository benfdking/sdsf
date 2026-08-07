package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func Test_data_source_user_test(t *testing.T) {

	// retrieve the token and user from env variables
	// required input variables
	slackAPIToken := os.Getenv("SLACK_API_TOKEN")
	slackUserID := os.Getenv("SLACK_USER_ID")

	if slackAPIToken == "" {
		t.Skip("SLACK_API_TOKEN environment variable not set, skipping test.")
	}

	if slackUserID == "" {
		t.Skip("SLACK_USER_ID environment variable not set, skipping test.")
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
                    terraform {
                        required_providers {
                            slack = {
                                source = "benfdking/slack"
                            }
                        }
                    }

                    provider "slack" {
                        api_token = var.slack_api_token
                    }

                    variable "slack_api_token" {
                        type        = string
                        description = "The API token for authenticating with Slack"
                        default = "%s"
                    }

                    variable "slack_user_id" {
                        type        = string
                        description = "The Slack user ID"
                        default = "%s"
                    }
												
                    data "slack_user" "test" {
                        id = var.slack_user_id
                    }

                    output "user" {
                        value = data.slack_user.test
                    }
                `, slackAPIToken, slackUserID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// Check the output result
						plancheck.ExpectKnownOutputValue("user", knownvalue.ObjectPartial(
							map[string]knownvalue.Check{
								"id": knownvalue.StringExact(slackUserID),
							},
						)),
					},
				},
				Check: resource.TestCheckFunc(func(s *terraform.State) error {
					// Retrieve the resource state
					rs, ok := s.RootModule().Resources["data.slack_user.test"]
					if !ok {
						return fmt.Errorf("resource not found: data.slack_user.test")
					}

					// Check the properties of the resource
					if rs.Primary.Attributes["id"] != slackUserID {
						return fmt.Errorf("expected id to be %s, got %s", slackUserID, rs.Primary.Attributes["id"])
					}

					return nil
				}),
			},
		},
	})
}
