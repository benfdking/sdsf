resource "slack_user_group" "example" {
  name        = "Marketing Team"
  description = "Marketing gurus, PR experts and product advocates."
  handle      = "marketing-team"
  channels    = ["open"]
}
