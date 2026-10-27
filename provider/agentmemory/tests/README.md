# Agent Memory tests

`unit/` contains external tests using the provider protocol or Terraform CLI.
API requests go to local HTTP test servers; these tests do not require Redis
Cloud credentials or create cloud resources.

The CLI tests exercise write-only model credentials. This package defaults
`TF_ACC_TERRAFORM_VERSION` to `1.11.1` and lets the test framework install that
version. Explicit `TF_ACC_TERRAFORM_PATH` or `TF_ACC_TERRAFORM_VERSION` settings
take precedence. The package-local default does not change the CLI used by other
resource packages or the shared CI environment.

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
