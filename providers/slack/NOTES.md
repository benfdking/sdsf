## Download Slack
curl -L -o /tmp/repo.zip https://github.com/tfstack/terraform-provider-slack-framework/archive/refs/heads/main.zip

## Setup Dev Environment
go mod edit -module terraform-provider-slack
go mod tidy

## Prepare Terraform for Local Provider Install
cat << EOF > ~/.terraformrc
provider_installation {
  dev_overrides {
      "benfdking/slack" = "/go/bin"
  }
  direct {}
}
EOF

## Verify the Initial Provider
go run main.go

## Locally Install Provider and Verify with Terraform
go install .

## Test
cd examples/provider-install-verification/
terraform plan
terraform -chdir=./examples/provider-install-verification plan
go test -v ./internal/provider/...
go test -v ./internal/provider/provider_test.go

### Run the Tests for the Package:
go test -v ./internal/provider

### Run a Specific Test File:
go test -v ./internal/provider/provider_test.go

### Run with Coverage:
go test -v -cover ./internal/provider

### Running Tests with Specific Test Names:
go test -v -run TestProviderConfigure ./internal/provider

### Running Acceptance Tests:
export SLACK_API_TOKEN="your_token_here"
export SLACK_USER_ID="replace_user"
export SLACK_USER_GROUP="replace_group"
export SLACK_DEFAULT_USER="replace_user"
export SLACK_USERS="user1,user2"
go test -v ./internal/provider
go test -v -cover ./internal/provider -run ^Test_resource_user_group$

### Running with Debug Logging:
TF_LOG=DEBUG go test -v ./internal/provider
