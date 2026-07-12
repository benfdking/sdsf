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

resource "cursor_automation" "daily_review" {
  name           = "Daily code review"
  model          = "gpt-5.5-high"
  enabled        = true
  memory_enabled = true

  prompt = <<-EOT
    Review recent changes, identify risks, and request the most appropriate reviewers.
  EOT

  triggers = [
    {
      cron = {
        cron = "0 9 * * 1-5"
      }
    }
  ]

  actions = [
    {
      requestReviewers = {}
    }
  ]

  git_config = {
    repositories = ["https://github.com/benfdking/sdsf"]
    branch       = "main"
  }
}

output "cursor_automation_id" {
  value = cursor_automation.daily_review.id
}
