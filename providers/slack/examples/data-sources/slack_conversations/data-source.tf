data "slack_conversations" "example" {
  exclude_archived = true
  types            = ["public_channel", "private_channel", "mpim", "im"]
}

output "example" {
  value = data.slack_conversations.example
}
