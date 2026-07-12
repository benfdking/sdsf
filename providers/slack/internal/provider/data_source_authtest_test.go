package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func Test_data_source_slack_authtest(t *testing.T) {

	// retrieve the token and user from env variables
	// required input variables
	slackAPIToken := os.Getenv("SLACK_API_TOKEN")

	if slackAPIToken == "" {
		t.Skip("SLACK_API_TOKEN environment variable not set, skipping test.")
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

                    data "slack_authtest" "test" {}

                    output "authtest" {
                        value = data.slack_authtest.test
                    }
                `, slackAPIToken),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
				},
				Check: resource.TestCheckFunc(func(s *terraform.State) error {
					// Retrieve the resource state
					rs, ok := s.RootModule().Resources["data.slack_authtest.test"]
					if !ok {
						return fmt.Errorf("resource not found: data.slack_authtest.test")
					}
					// Check the properties of the resource
					if _, ok := rs.Primary.Attributes["team"]; !ok {
						return fmt.Errorf("expected team to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["team_id"]; !ok {
						return fmt.Errorf("expected team_id to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["user"]; !ok {
						return fmt.Errorf("expected user to exist in attributes, but it does not")
					}
					if _, ok := rs.Primary.Attributes["user_id"]; !ok {
						return fmt.Errorf("expected user_id to exist in attributes, but it does not")
					}
					return nil
				}),
			},
		},
	})
}
