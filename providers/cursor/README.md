# Terraform provider for Cursor Automations

This provider manages Cursor Automations as first-class Terraform resources. Its API client and reconciliation primitives originated in `incident-io/internal-ai/internal/cursorautomations` at revision `e7f61de25ad26fd0cca52671c3abbbd5b7b026af`.

It uses Cursor's unofficial dashboard API. The API or its authentication mechanism may change without notice.

## Authentication

Set credentials in the provider block or through environment variables:

- `CURSOR_SESSION_TOKEN`: the `WorkosCursorSessionToken` cookie value
- `CURSOR_TEAM_ID`: the Cursor team ID

The session token is sensitive. Do not commit it to Terraform configuration.

## Example

```hcl
resource "cursor_automation" "review" {
  name           = "Daily code review"
  model          = "gpt-5.5-high"
  enabled        = true
  memory_enabled = true
  prompt         = "Review recent changes and request the right reviewers."

  triggers = [{
    cron = { cron = "0 9 * * 1-5" }
  }]

  actions = [{
    requestReviewers = {}
  }]

  git_config = {
    repositories = ["https://github.com/benfdking/sdsf"]
    branch       = "main"
  }
}
```

`triggers` and `actions` are native HCL values whose object keys mirror Cursor's automation payload. Integration-specific fields remain typed Terraform values rather than YAML or embedded JSON.

Existing automations are adopted by matching `name` during creation. Cursor does not publish an Automations management API, and the copied dashboard client has no verified delete operation, so destroying a Terraform resource removes it from Terraform state but leaves the live Cursor automation in place.

See [`examples/basic`](examples/basic) for a complete example.

## Build and test

```bash
go build ./...
go test ./...
```
