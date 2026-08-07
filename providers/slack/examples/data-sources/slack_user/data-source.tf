# list a user
data "slack_user" "example" {
  id = "U0123456789"
}

output "example" {
  value = data.slack_user.example
}
