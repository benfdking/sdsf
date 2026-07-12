data "slack_user_profile" "example" {
  id = "U0123456789"
}

output "example" {
  value = data.slack_user_profile.example
}
