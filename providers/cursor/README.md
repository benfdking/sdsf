# Terraform provider for Cursor Automations

This provider wraps the directory-based Cursor Automation reconciliation code copied from `incident-io/internal-ai/internal/cursorautomations` at revision `e7f61de25ad26fd0cca52671c3abbbd5b7b026af`.

It uses Cursor's unofficial dashboard API. The API or its authentication mechanism may change without notice.

## Authentication

Set credentials in the provider block or through environment variables:

- `CURSOR_SESSION_TOKEN`: the `WorkosCursorSessionToken` cookie value
- `CURSOR_TEAM_ID`: the Cursor team ID

The session token is sensitive. Do not commit it to Terraform configuration or state.

## Automation directory

Each direct subdirectory contains an `automation.yaml` and `prompt.md`:

```text
automations/
  daily-review/
    automation.yaml
    prompt.md
```

The `cursor_automation_sync` resource reconciles the complete directory. Removing the Terraform resource stops reconciliation but does not delete live Cursor automations because the copied client has no delete endpoint.

See [`examples/basic`](examples/basic) for a complete example.

## Build and test

```bash
go build ./...
go test ./...
```
