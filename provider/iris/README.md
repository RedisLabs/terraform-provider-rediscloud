# Iris resources

Each resource uses an explicit Terraform Plugin Framework model, REST request and
response structs, and CRUD methods. There is no runtime API specification,
metadata interpreter, reflection-based mapping or custom code generation step.

- `resource_<name>_schema.go` declares its Terraform schema and model.
- `resource_<name>.go` implements resource lifecycle methods.
- `memory_blocks.go` and `identity_grants.go` define typed native-set elements.
- `identity_grants.go` defines REST grant structs and their explicit conversion. API keys require at least one grant and expose no JSON or
  constraints configuration. Grant and action sets are order-independent.
- `memory_api.go` contains typed API structures. `memory_patch.go` compares the
  Terraform plan and
  prior state to build PATCH bodies directly; configuration supplies write-only
  keys. `memory_state.go` maps authoritative GET responses by setting group.
  Mutation responses supply creation identifiers and one-time tokens; memory
  types also retain their complete creation response if a follow-up GET fails.
  Collections use Framework
  `types.Set`, `ElementsAs` and `SetValueFrom`; null, unknown and empty values
  remain distinct.
- Resource schemas pair write-only model keys with rotation versions; Iris
  validates provider-specific credential requirements. Plan checks require
  explicit creation flags and complete model selections while allowing updates
  to inherit known settings. Resource plan methods reject
  unsafe same-name edits because the service permanently retires removed names.
- `helpers.go` contains small value/configuration helpers and the constructor list.
- `validators.go` checks safe, opaque path identifiers. Schemas retain nonempty
  values, related-field rules and positive credential versions; the service owns
  supported options, ID formats and upper limits.
- `provider/client/iris.go` is the HTTP transport; `Do` accepts typed bodies and
  responses.

Schemas and models were initially scaffolded with HashiCorp's generator. They
are ordinary maintained Go code now. Edit the resource's schema/model and CRUD
mapping together; no generation command is needed. All settings use native
Terraform attributes and blocks. JSON serialization is confined to the HTTP API.
Resource pages are generated from schema descriptions and the HCL/import examples
under `examples/`. Regenerate all three Iris pages from the repository root:

```sh
nix develop --command make docs-iris
```

The target renders into ignored `bin/iris-docs/` and copies only the Iris pages
to the repository's existing `docs/resources/rediscloud_iris_*.md` paths. Edit
descriptions or examples instead of editing the generated Markdown.

Run checks from the repository checkout using its Nix environment:

```sh
nix develop --command go test ./provider/iris/... ./provider/client
nix develop --command go build ./...
nix develop --command go vet ./...
nix develop --command golangci-lint fmt --diff
```

The local Terraform lifecycle test exercises all three resources through the mux.
Direct unit tests cover schema/model compatibility, typed mappings, secrets,
not-found handling and state recovery after a failed create/readiness check.
Live Agent Memory tests and their credential requirements are documented in
`tests/acceptance/README.md`. Model-provider secrets are optional and never
checked into the fixtures.

Inline create-only type snapshots and credential versions cannot be recovered
through import. Write-only model keys come from configuration and are never
retained in plan or state. Memory-type identifiers include both store and name.

Identity API keys cannot be imported because their tokens are returned only at
creation. The local Terraform suite verifies that import is rejected.
Replacement uses a second key resource: create it, switch consumers, then remove
the old key. The lifecycle test checks this handover with distinct IDs and tokens.
