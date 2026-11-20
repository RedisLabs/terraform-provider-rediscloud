# Iris schema generation

Run from the provider checkout using its pinned Nix environment:

```sh
nix develop --command bash scripts/iris-codegen/generate.sh
nix develop --command go test ./internal/irisgen ./provider/iris/...
```

The pipeline uses HashiCorp OpenAPI generator v0.3.0 and Framework generator
v0.4.1. It has no dependency on a local API-client checkout or uncommitted code.
The input snapshots and their upstream revision are recorded in `inputs/`.

`internal/irisgen` prepares operation-level schemas, then adapts the resulting
provider code specifications. Output is committed under `provider/iris/`:

- `generated/` contains the Framework generator's schemas and models.
- `definitions_gen.json` maps Terraform attributes to API properties and records
  request/response shapes, paths, mutability and revision requirements.
- `schemas_gen.go` connects each definition to its generated schema function.

Preparation represents complex API properties as JSON strings. This preserves
untyped `Grant.constraints` and deeply nested product grant maps, avoids the
Framework generator's repeated leaf-type names, and serves protocol v5 without
nested attributes. Runtime validators still check the original request shape.
Password-bearing JSON remains sensitive. Original input files are unchanged.
The preparation step translates Smithy-generated regex escapes to Go syntax;
path validators also reject dot segments and invalid tenant components.
Unknown regex syntax fails generation.

The adaptation step derives configuration flags from the create operation,
makes required path parameters configurable, treats server revisions as
read-only, and assigns replacement rules to immutable fields. Optional computed
fields require replacement only when explicitly configured. It verifies all six
expected resources were emitted, so a warning-only skipped resource fails the
pipeline. Generated files are formatted with the repository's import grouping.

The handwritten lifecycle in `provider/iris` supplies authenticated HTTP calls,
JSON mapping, secret retention, revision checks, import and readiness handling.
These behaviours cannot be inferred from an OpenAPI schema alone. The provider
registers these resources separately from the existing CAPI implementations.
See `docs/guides/iris.md` for public usage and limitations.

Tests cover real Terraform CRUD/import flows through the mux against local HTTP
servers, schema/model equality, JSON validation, mutation recovery, secret
retention, exact numbers, credential isolation and HTTP failures. Live Agent Memory acceptance against app-qa is documented in
`provider/iris/tests/acceptance/README.md`; customer-model tests require separate
model-provider credentials.

The original unadapted experiment is retained in `experiments/openapi-codegen/`.
Its failures describe raw generator output, not the integrated resources.
