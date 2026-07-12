# Provenance

## Slack provider

- Source: `tfstack/terraform-provider-slack`
- Source revision: `87fa7c87cb7124ee50cf40f3081668b693e81855`
- License: MIT; the original notice is preserved in [`providers/slack/LICENSE`](providers/slack/LICENSE)
- Local changes: module and provider addresses moved to `benfdking/sdsf`, dependencies upgraded, Slack's `Has2FA` pointer API adopted, and network-dependent HTTP tests replaced with deterministic local tests

## Cursor Automations

- Source: `incident-io/internal-ai/internal/cursorautomations`
- Source revision: `e7f61de25ad26fd0cca52671c3abbbd5b7b026af`
- The package was initially copied byte-for-byte into [`providers/cursor/internal/cursorautomations`](providers/cursor/internal/cursorautomations), then given an exported constructor and Git configuration type so Terraform resources can construct automations directly
- No repository-level open-source license was present in the private source repository at the copied revision
- A Terraform Plugin Framework wrapper exposes each automation as a `cursor_automation` resource; the original YAML-directory sync resource is not used

Keep this repository private unless the owners of all included code explicitly authorize redistribution.
