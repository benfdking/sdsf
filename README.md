# SDSF — Software Defined Software Factory

SDSF is a private monorepo for Terraform providers that manage the tools used by a software-defined software factory.

## Providers

| Provider | Purpose | Location |
| --- | --- | --- |
| Slack | Slack users, groups, status, conversations, and HTTP helpers | [`providers/slack`](providers/slack) |
| Cursor | First-class Terraform resources for Cursor Automations | [`providers/cursor`](providers/cursor) |
| Linear | Team and project lookup, issue labels, and custom views | [`providers/linear`](providers/linear) |

All providers use the Terraform Plugin Framework and require Go 1.26 or newer.

```bash
go test ./providers/slack/...
go test ./providers/cursor/...
go test ./providers/linear/...
```

## Licensing and provenance

This repository has no repository-wide open-source license because it combines code with different ownership and licensing terms. See [`PROVENANCE.md`](PROVENANCE.md) and the provider-specific notices before redistributing any part of it.
