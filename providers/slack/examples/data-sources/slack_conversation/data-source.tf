data "slack_conversation" "example" {
  name             = "open"
  exclude_archived = true
  types            = ["public_channel"]
  query_limit      = 1000
}

output "example" {
  value = data.slack_conversation.example
}
