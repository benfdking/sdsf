# SDSF — Software Defined Software Factory

SDSF is a private monorepo for the Terraform providers and command line tools that manage a software-defined software factory.

## Providers

| Provider | Purpose | Location |
| --- | --- | --- |
| Slack | Slack users, groups, status, conversations, and HTTP helpers | [`providers/slack`](providers/slack) |
| Cursor | First-class Terraform resources for Cursor Automations | [`providers/cursor`](providers/cursor) |
| Linear | Team and project lookup, issue labels, and custom views | [`providers/linear`](providers/linear) |

All providers use the Terraform Plugin Framework and require Go 1.26 or newer.

## Command line tools

| Tool | Purpose | Location |
| --- | --- | --- |
| `cursorjob` | Submit jobs to Cursor's Cloud Agents API, then attach and block until they finish | [`cli/cursorjob`](cli/cursorjob) |

## Testing

Each directory is its own Go module, tied together by [`go.work`](go.work).

```bash
go test ./providers/slack/...
go test ./providers/cursor/...
go test ./providers/linear/...
go test ./cli/cursorjob/...
```

## Licensing and provenance

This repository has no repository-wide open-source license because it combines code with different ownership and licensing terms. See [`PROVENANCE.md`](PROVENANCE.md) and the provider-specific notices before redistributing any part of it.
