# Terraform provider for Slack

This provider manages Slack users and user groups and exposes Slack data sources and an HTTP helper function. It is based on `tfstack/terraform-provider-slack` revision `87fa7c87cb7124ee50cf40f3081668b693e81855` and has been updated to current Go, Terraform Plugin Framework, and `slack-go/slack` APIs.

The original MIT license and copyright notice are preserved in [`LICENSE`](LICENSE).

## Requirements

- Terraform 1.0 or newer
- Go 1.26 or newer
- A Slack token provided through the provider's `token` attribute or `SLACK_TOKEN`

## Build and test

```bash
go build ./...
go test ./...
```

Provider documentation is available under [`docs`](docs).
