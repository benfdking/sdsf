# Terraform provider for Linear

This provider manages shared Linear workspace configuration through Linear's public GraphQL API.

## Authentication

Use a personal API key:

```bash
export LINEAR_API_KEY="lin_api_..."
```

Alternatively, set `LINEAR_ACCESS_TOKEN` to an OAuth access token. The corresponding `api_key` and `access_token` provider attributes are also available; configure only one authentication method.

## Resources and data sources

- `linear_team` looks up a team by UUID or key.
- `linear_project` looks up a project by UUID or URL slug ID.
- `linear_issue_label` manages team-scoped and workspace-scoped issue labels.
- `linear_custom_view` manages custom issue views.

Custom views accept `filter_json`, a JSON-encoded Linear `IssueFilter`. Keeping the filter as JSON lets configurations use new Linear filter fields without waiting for a provider schema update.

```hcl
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
```

See [`examples/basic`](examples/basic) for a complete configuration.

## Build and test

```bash
go build ./...
go test ./...
```
