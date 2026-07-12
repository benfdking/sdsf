data "slack_authtest" "example" {}

output "example" {
  value = data.slack_authtest.example
}
