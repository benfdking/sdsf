# list all users
data "slack_user_groups" "example"{
}

output "example" {
  value = data.slack_user_groups.example
}
