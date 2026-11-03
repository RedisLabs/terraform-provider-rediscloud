# Agent Memory tests

`unit/` contains external tests using the provider protocol or Terraform CLI.
API requests go to local HTTP test servers; these tests do not require Redis
Cloud credentials or create cloud resources.

Run the CLI tests in the repository's Nix development shell, which pins Terraform
to 1.16.7 for both local development and CI, including write-only credential tests.

The Agent Memory feature minimum is Terraform 1.11.1. To verify that minimum
without changing the development toolchain, use a separate CLI binary:

```sh
TF_ACC_TERRAFORM_PATH=/path/to/terraform-1.11.1 go test ./provider/agentmemory/...
```

Terraform 1.11.0 fails the credential lifecycle test when values are both
ephemeral and sensitive. The serialization fix is documented in the
[Terraform 1.11.1 changelog](https://github.com/hashicorp/terraform/blob/v1.11.1/CHANGELOG.md).

Reusable lifecycle configurations live in `unit/testdata`. Load them with
`config.StaticFile` and keep `ConfigVariables` beside each test step. Acceptance
tests load the shared infrastructure and selected store fixture with
`config.StaticDirectory`, with inputs declared in each step as well.

Tests that access private resource types or helpers remain in the parent
`agentmemory` package. Go subpackages cannot access those private symbols.

Run both groups from the repository root:

```sh
go test ./provider/agentmemory/...
```

The repository-wide `go test ./...` command also includes both groups.
To include the resource package in coverage collected from the external tests:

```sh
mkdir -p bin
go test -coverpkg=./provider/agentmemory -coverprofile=bin/agent-memory-coverage.out ./provider/agentmemory/...
```

[`acceptance/`](acceptance/README.md) contains tests against real Redis Cloud
resources, following the repository's `TestAcc` naming and `TF_ACC` conventions.
