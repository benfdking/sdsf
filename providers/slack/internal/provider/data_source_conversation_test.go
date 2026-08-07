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

func Test_data_source_conversation_test(t *testing.T) {

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

                    data "slack_conversation" "test" {
												name = "open"
                    }

                    output "conversation" {
                        value = data.slack_conversation.test
                    }
                `, slackAPIToken),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
				},
				Check: resource.TestCheckFunc(func(s *terraform.State) error {
					// Retrieve the resource state
					_, ok := s.RootModule().Resources["data.slack_conversation.test"]
					if !ok {
						return fmt.Errorf("resource not found: data.slack_conversation.test")
					}

					return nil
				}),
			},
		},
	})
}
