resource "slack_user_status" "wfh" {
  id = "U0123456789"
  status_text  = "Working from home"
  status_emoji = ":house_with_garden:"
  status_expiration = 1728161574
}

resource "slack_user_status" "busy" {
  id = "U0123456789"
  status_text  = "Busy"
  status_emoji = ":red_circle:"
  status_expiration = 1728161574
}
