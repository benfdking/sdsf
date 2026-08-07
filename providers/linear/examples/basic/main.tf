terraform {
  required_providers {
    linear = {
      source = "benfdking/linear"
    }
  }
}

provider "linear" {
  # Defaults to LINEAR_API_KEY or LINEAR_ACCESS_TOKEN.
}

data "linear_team" "engineering" {
  key = "ENG"
}

data "linear_project" "launch" {
  slug_id = "launch-abc123"
}

resource "linear_issue_label" "security" {
  name        = "Security"
  description = "Security-related work"
  color       = "#EB5757"
  team_id     = data.linear_team.engineering.id
}

resource "linear_custom_view" "security" {
  name        = "Open security work"
  description = "Security issues that are not completed"
  team_id     = data.linear_team.engineering.id
  shared      = true
  color       = "#EB5757"
  icon        = "Shield"

  filter_json = jsonencode({
    labels = {
      some = {
        id = { eq = linear_issue_label.security.id }
      }
    }
    completedAt = { null = true }
  })
}
