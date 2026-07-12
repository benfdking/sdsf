terraform {
  required_providers {
    cursor = {
      source = "benfdking/cursor"
    }
  }
}

provider "cursor" {
  # Defaults to CURSOR_SESSION_TOKEN and CURSOR_TEAM_ID.
}

resource "cursor_automation_sync" "factory" {
  automations_directory = "${path.module}/automations"
}

output "cursor_sync_summary" {
  value = {
    created   = cursor_automation_sync.factory.created
    updated   = cursor_automation_sync.factory.updated
    unchanged = cursor_automation_sync.factory.unchanged
  }
}
