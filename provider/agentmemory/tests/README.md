# Agent Memory tests

`unit/` contains external tests using the provider protocol or Terraform CLI.
API requests go to local HTTP test servers; these tests do not require Redis
Cloud credentials or create cloud resources.

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
