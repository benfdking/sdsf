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

func Test_data_source_slack_user_profile(t *testing.T) {

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
												
                    data "slack_user_profile" "test" {
                        id = var.slack_user_id
                    }

                    output "user_profile" {
                        value = data.slack_user_profile.test
                    }
                `, slackAPIToken, slackUserID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// Check the output result
						plancheck.ExpectKnownOutputValue("user_profile", knownvalue.ObjectPartial(
							map[string]knownvalue.Check{
								"id": knownvalue.StringExact(slackUserID),
							},
						)),
					},
				},
				Check: resource.TestCheckFunc(func(s *terraform.State) error {
					// Retrieve the resource state
					rs, ok := s.RootModule().Resources["data.slack_user_profile.test"]
					if !ok {
						return fmt.Errorf("resource not found: data.slack_user_profile.test")
					}
					// Check the properties of the resource
					if rs.Primary.Attributes["id"] != slackUserID {
						return fmt.Errorf("expected id to be %s, got %s", slackUserID, rs.Primary.Attributes["id"])
					}
					if _, ok := rs.Primary.Attributes["api_app_id"]; !ok {
						return fmt.Errorf("expected api_app_id to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["avatar_hash"]; !ok {
						return fmt.Errorf("expected avatar_hash to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["bot_id"]; !ok {
						return fmt.Errorf("expected bot_id to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["display_name"]; !ok {
						return fmt.Errorf("expected display_name to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["display_name_normalized"]; !ok {
						return fmt.Errorf("expected display_name_normalized to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["email"]; !ok {
						return fmt.Errorf("expected email to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["first_name"]; !ok {
						return fmt.Errorf("expected first_name to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["image_192"]; !ok {
						return fmt.Errorf("expected image_192 to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["image_24"]; !ok {
						return fmt.Errorf("expected image_24 to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["image_32"]; !ok {
						return fmt.Errorf("expected image_32 to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["image_48"]; !ok {
						return fmt.Errorf("expected image_48 to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["image_512"]; !ok {
						return fmt.Errorf("expected image_512 to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["image_72"]; !ok {
						return fmt.Errorf("expected image_72 to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["image_original"]; !ok {
						return fmt.Errorf("expected image_original to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["last_name"]; !ok {
						return fmt.Errorf("expected last_name to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["phone"]; !ok {
						return fmt.Errorf("expected phone to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["real_name"]; !ok {
						return fmt.Errorf("expected real_name to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["real_name_normalized"]; !ok {
						return fmt.Errorf("expected real_name_normalized to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["skype"]; !ok {
						return fmt.Errorf("expected skype to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["status_emoji"]; !ok {
						return fmt.Errorf("expected status_emoji to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["status_expiration"]; !ok {
						return fmt.Errorf("expected status_expiration to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["status_text"]; !ok {
						return fmt.Errorf("expected status_text to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["team"]; !ok {
						return fmt.Errorf("expected team to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["title"]; !ok {
						return fmt.Errorf("expected title to exist in attributes, but it does not")
					}
					return nil
				}),
			},
		},
	})
}
