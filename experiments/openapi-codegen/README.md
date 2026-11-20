# OpenAPI generator experiment

This is the original baseline. Separate, integrated Iris resources now live in
`provider/iris/`; see `scripts/iris-codegen/README.md` and `docs/guides/iris.md`.
The failures below refer to the unadapted generator output retained here.

Tried HashiCorp's OpenAPI generator v0.3.0 and Framework generator v0.4.1
against real Agent Memory and Identity Service specifications on 2026-11-19.
Base: `release/2.21.0` at `874e846`.

The tools produce useful schema/model starting points, but the output is not
ready to register in this provider. No production resources are changed.
`_generated/` and `_probe/` are excluded from normal Go `./...` traversal.
Generated files are retained verbatim, including errors.

## Reproduce

Run from this worktree with the project's pinned Nix environment:

```sh
nix develop --command bash experiments/openapi-codegen/generate.sh
nix develop --command bash experiments/openapi-codegen/verify.sh
```

Go runs the generators at explicit versions without changing the provider's
module dependencies. Inputs are saved with checksums and source provenance.
Configurations specify CRUD operations, exclude the unsupported memory
`projectId` parameter, and alias path parameters. The verification script
checks resource counts because the OpenAPI generator can skip a whole resource
with a warning and still exit zero. It records compilation failure, then runs
the schema probe on packages that compile. The probe prints diagnostics and
recovers panics to examine every schema; a zero exit is not a compatibility pass.

## Observed results

| Resource | Generation | Compile | Schema construction / protocol 5 |
| --- | --- | --- | --- |
| Memory store | Yes | Fails | Not reached |
| Memory extractor | Yes | Passes | Passes |
| Memory type | Yes | Passes | Protocol 5 conversion fails |
| Identity API key, baseline | Skipped | No output | No output |
| Identity API key, adapted | Yes | Passes | Panics |
| Identity provider | Yes | Passes | Protocol 5 conversion fails |
| Identity subject grant set | Yes | Passes | Panics |

Evidence is retained in `results/compile.log`, `results/schema-probe.log`,
per-service generator logs and provider code specifications.

- **Duplicate Go names:** memory store output declares `CredentialsType`,
  `CredentialsValue` and their helpers repeatedly for nested model credentials.
  The Framework generator uses leaf attribute names for these types.
- **Untyped JSON:** `Grant.constraints` has only a description and no type.
  The baseline OpenAPI generator skips the entire API key resource. The separate
  `identity-adapted.yml` excludes `grants.constraints`, demonstrating generation
  at the cost of losing that API capability. It is not a production workaround.
- **Regex dialect:** generated API key and subject grant set schemas call
  `regexp.MustCompile` on a tenant pattern containing `\u` escapes, which Go's
  regex engine rejects. Compilation alone does not reveal the panic.
- **Protocol:** generated `SingleNestedAttribute` / `ListNestedAttribute`
  structures cannot be served through this provider's protocol-v5 mux. The
  probe demonstrates conversion errors for memory type `extraction_strategy`
  and identity provider `token_exchange`.
- **Identifier aliases:** aliases affect operation parameters, not all response
  properties. Output still contains both `id` and `store_id` for memory stores,
  and both `id` and `key_id` for API keys.
- **Path parameter semantics:** identity provider `tenant` is generated as
  computed even though creation needs it as a path parameter. Review required,
  optional and computed flags against actual operations before adoption.
- **Read models:** subject grant set `products` becomes a map of empty objects,
  losing useful product-grant structure without a warning.

The API key token and memory model API keys are correctly marked sensitive
from the OpenAPI password format. Enum, length and numeric validators are also
emitted; these remain useful starting points after correcting the above issues.

## Implications for this provider

The memory spec describes `/iris/v1/agent-memory/stores`; the existing resource
calls CAPI `/memory-stores` through `rediscloud-go-api`. It also exposes a
flattened Terraform shape (for example `database_id`, TTL fields and model
blocks), while generated output follows nested API objects. The generic
Identity API key API differs from the store-scoped Agent Memory API key resource.
This experiment does not establish schema or lifecycle compatibility with either
existing resource.

These generators supply schemas, models and conversion helpers. CRUD clients,
asynchronous task polling, readiness, import, state handling, recovery,
replacement rules and cross-field validation still need implementation.
The probe uses inert resource stubs solely to request a protocol-v5 schema;
no API calls or infrastructure changes are made.

Before adoption, fix or upstream the repeated-name and regex problems, model
dynamic grants explicitly, choose protocol-compatible nested blocks, and map the
actual supported API and Terraform contract. An API-spec change alone cannot
express the existing resource lifecycle.

References:

- [OpenAPI generator](https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator)
- [Framework generator](https://developer.hashicorp.com/terraform/plugin/code-generation/framework-generator)
