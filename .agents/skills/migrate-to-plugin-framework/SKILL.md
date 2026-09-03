---
name: migrate-to-plugin-framework
description: >-
  Migrate Redis Cloud Terraform provider data sources and resources from
  terraform-plugin-sdk/v2 to terraform-plugin-framework while preserving the
  protocol-v5 schema and existing behaviour. Use for SDKv2-to-Framework
  migrations, new Framework resources or data sources, mux registration,
  Framework schemas and models, CRUD or state conversion, API-to-Framework
  mapping helpers including legacy flatten helpers, provider custom types,
  migration acceptance tests, and manual
  migration verification.
---

# Migrate to the Plugin Framework

Treat a migration as a compatibility change. Move ownership from the SDKv2
provider to the Framework provider without changing the public Terraform shape
or the values users already have in state.

Read `AGENTS.md` first for repository commands, branch policy, service ownership,
and manual-provider setup.

## Establish the baseline

1. Read the complete SDKv2 implementation, its tests, and every helper it calls.
   Use them as the behavioural specification rather than relying on the migration
   ticket.
2. Compare the schema field by field. Record names, types, collection kinds,
   cardinality, validators, defaults, conflicts, and Required, Optional, and
   Computed combinations.
3. Search the provider for sibling consumers of every nested shape and shared
   helper. A data source may expose a computed copy of a block that users
   configure on its resource.
4. Verify that files expected from a parent migration branch are present. Base a
   stacked migration on that parent rather than recreating its helpers.
5. Ask before deliberately changing observable behaviour. Fixing an apparent
   SDKv2 bug is separate from faithfully migrating it, even when the proposed fix
   is non-breaking.

## Preserve the protocol-v5 schema

The provider mux speaks protocol v5. Translate SDKv2 collection shapes according
to what they represent:

- Convert `TypeList` or `TypeSet` with `Elem: &schema.Resource{}` to
  `schema.ListNestedBlock` or `schema.SetNestedBlock`.
- Do not use Framework nested attributes for those object collections. Nested
  attributes require protocol v6 and can panic when served through this mux.
- Convert primitive lists and sets to `schema.ListAttribute` and
  `schema.SetAttribute` with an `ElementType`.
- Convert primitive maps to `schema.MapAttribute`.

Preserve the public block and attribute names, nesting, collection kind, and
configuration semantics exactly. Similar subscription types do not necessarily
give a field the same Required, Optional, or Computed flags, so verify each
schema independently.

Keep descriptions meaningful in code you touch. Update generated or published
documentation only when the schema or its wording actually changes.

## Place and register the implementation

Put a migrated implementation in its resource-family package. Use a focused file
layout unless the package already has an established equivalent:

- `datasource_<name>.go`, `datasource_<name>_model.go`, and
  `datasource_<name>_read.go` for a data source.
- `resource_<name>.go`, `resource_<name>_model.go`, and
  `resource_<name>_crud.go` for a resource.

Add compile-time assertions for every Framework interface the implementation
supports. Configure the API client through `Configure`, and derive the Terraform
type name from `req.ProviderTypeName` plus the public suffix.

Register the constructor in `provider/framework_provider.go`. Remove the SDKv2
registration from `provider/sdk_provider.go` and leave its existing Framework
ownership marker. Delete the old SDKv2 implementation only after searching for
remaining callers of its helpers.

Framework resources and data sources do not receive an implicit `id` attribute.
For a migrated implementation, declare a computed string `id` as the SDKv2
implementation exposed one implicitly; existing configurations and state may reference it.
For a new implementation with no SDKv2 predecessor, add `id` only when the value
has a useful domain meaning.

## Map values without changing state semantics

Match the value written by the SDKv2 code, including its treatment of missing API
fields:

- Preserve empty strings when SDKv2 wrote `""`. Do not replace them with null
  simply because pointer-based Framework constructors make that convenient.
- Use a pointer constructor only when SDKv2 genuinely left the attribute absent.
- Preserve zero values for integer fields when that was the previous behaviour.
- Keep existing special defaults, such as a boolean that defaults to true when
  the API omits it.
- Prefer constants from `rediscloud-go-api` to duplicated status or enum strings.

Acceptance checks alone do not prove empty-string compatibility. Terraform's
testing helpers can treat an absent value as equivalent to `""` in common
resource-attribute checks. Inspect the SDKv2 assignment and use a state check
that distinguishes known empty strings from null when the distinction matters.

Use the project marker `//TODO(TF3.0) make <field> nullable` for all fields kept
as an empty string solely for backward compatibility and expected to become null
in the next major version.

When a resource field is supplied by configuration but never returned by the
API, leave its existing model value untouched during reads. Assigning a null
value erases configuration that SDKv2 previously retained in state.

## Name mapping helpers by result and direction

Rename an SDKv2-style `flattenX` helper when a migration moves, replaces, or
otherwise changes it. Do not introduce new `flatten` names in Framework code.

Name an API-to-Framework collection mapper
`<Domain><TerraformShape>FromAPI`. Include `List`, `Set`, `Map`, or `Object` in
the name so the result is clear at the call site. For example, use
`PricingListFromAPI` and `ResourceTagsMapFromAPI`.

Keep `NewX` for a real constructor that returns the named custom type or value it
creates. `NewMaintenanceList`, for example, constructs a
`MaintenanceListValue`; it is not an ordinary mapper returning `types.List`.

Update callers, comments, and test names with the helper. Limit the rename to
helpers touched by the migration rather than sweeping unrelated legacy code.

## Choose a strategy for nested collections

Make the decision from every resource and data source that uses the shape, not
only from the schema currently being migrated.

### Configurable shapes

Create a reusable Framework custom collection type when the nested shape is user configurable somewhere. Required, Optional, or Optional-and-Computed blocks fall
into this category. Maintenance windows are the representative subscription
shape: a data source reads them as computed data, while subscription resources
accept them as configuration.

Apply this decision recursively to nested object collections. If a configurable
custom block contains another configurable list or set of objects, give the
nested collection its own custom type so the parent model exposes a typed value
with its own `AsModels`. Keep primitive collections as ordinary Framework values
unless they need additional domain behaviour.

Introduce the custom type in whichever migration reaches the shared shape first.
Prefer introducing it with the data-source migration when possible. The later
resource migration can then consume the same typed value and avoids carrying the
type implementation in its own already-large diff.

Use a custom list type when callers need the whole list as a domain value:

1. Define `XListType` and `XListValue` by embedding `basetypes.ListType` and
   `basetypes.ListValue`.
2. Implement the Framework conversion, equality, and type methods required by
   `basetypes.ListTypable` and `basetypes.ListValuable`. Add compile-time
   interface assertions.
3. Attach the type to `schema.ListNestedBlock.CustomType` and use `XListValue` in
   the containing `tfsdk` model.
4. Represent element data with `tfsdk`-tagged model structs. Construct the
   collection from those models as described in the shared rules below.
5. Add `AsModels(ctx) ([]XModel, diag.Diagnostics)` to the list value and decode
   through the embedded `ElementsAs` method.

Do not override `Elements()`. Its signature belongs to the Framework value
interface and must continue to return `[]attr.Value`. Do not add a custom element
type merely to avoid casts in provider code. `AsModels` supplies typed access to
the complete collection while ordinary object elements retain standard Framework
behaviour. Callers that need one element can index the returned model slice.

Require callers to check `IsNull()` and `IsUnknown()` before using `AsModels`, and
propagate its diagnostics. Constructors must preserve null and unknown states
rather than converting them to empty known lists.

### Strictly computed shapes

Keep ordinary Framework collection values when a nested shape is computed in
every resource and data source. Pricing is the representative subscription
shape. It is returned by the API and is never user configuration, so a provider
custom type adds no useful configuration or CRUD boundary.

Still give the element a `tfsdk`-tagged model such as `PricingModel`. Derive the
element attribute map with `customtypes.AttrTypesOf(PricingModel{})`. This keeps
the model as the source of attribute names and types so mapping code does not
repeat hardcoded string keys. Store the result in a normal `types.List` or
`types.Set` model field.

`AttrTypesOf` expects model fields that implement `attr.Value`. A zero
`types.String`, `types.Int64`, or similar scalar can report its type directly. A
list or set field must be seeded with its element type because an uninitialised
collection cannot describe that element type.

### Rules shared by both strategies

- Map API values into a non-nil model slice such as
  `make([]PricingModel, 0, len(apiValues))`, then convert the complete slice with
  the matching whole-collection function, such as `types.ListValueFrom` or
  `types.SetValueFrom`. Prefer this to converting each model with
  `types.ObjectValueFrom` and manually assembling `[]attr.Value`. The
  whole-collection function centralises conversion diagnostics, and the non-nil
  empty slice preserves a known empty Terraform collection when the API returns
  no values. Use per-element conversion only when an element needs custom logic
  or element-specific diagnostic handling.
- Sort API results before creating a Terraform list when the API does not
  guarantee order. Use a composite key containing every field needed to produce
  a stable result.
- Preserve null, unknown, empty, and known states deliberately.
- Keep the schema's nested object attributes and the model-derived type map in
  sync. This consistency is checked at runtime, so cover it with tests.
- Return diagnostics from Framework conversion functions immediately when they
  contain errors.

For a custom collection, test type equality, Terraform conversion, null and
unknown handling, API-to-value construction, and `AsModels`. For a computed-only
collection, test model decoding, empty and nil responses, value compatibility,
and deterministic ordering.

## Share and retire helpers carefully

Put Framework helpers used by more than one package in `provider/utils`, and put
reusable Framework custom types in `provider/customtypes`. Reuse an existing
helper only after confirming that its null handling, ordering, and element shape
match the migration.

Search the entire provider before removing an SDKv2 helper or constant. Other
SDKv2 resources may still require its map-based return value even when Framework
code has moved to `attr.Value`. If a remaining caller can use an identical shared
helper, repoint that caller and then remove the duplicate. Otherwise leave the
SDKv2 helper in place until its final caller migrates and just add a reminder
comment for that.

## Preserve Framework resource lifecycle behaviour

Framework request and response types prevent `Create` or `Update` from directly
delegating to `Read`. Factor shared API-to-model logic into an internal helper
that all three operations can call.

Apply these resource rules:

- Treat NotFound during `Read` as a missing resource and remove it from state.
- Continue to propagate NotFound from `Delete` unless the existing resource
  deliberately does otherwise. These resources assume Terraform owns their
  lifecycle.
- Continue using `terraform-plugin-sdk/v2/helper/retry` for state polling. There
  is no need to replace a working `retry.StateChangeConf` merely because the
  resource itself uses the Framework.
- If creation or update produces a remote object and a later readiness wait
  fails, write its identifier and last observed status to state before returning
  the error. Otherwise Terraform loses ownership of an object it created.

For imports, implement `ResourceWithImportState` and verify any configuration-only
secret fields through the same ignore behaviour used by the existing acceptance
test.

## Update tests with the migration

Add table-driven unit tests for pure filters, mapping helpers, sort keys, custom
types, and model conversions. Keep those tests independent of Terraform and live
API credentials where possible.

Acceptance tests use `terraform-plugin-testing` and run against a real account
only with `TF_ACC=1`. Keep these common pieces unless the existing test requires
more:

- `testhelpers.ProtoV5ProviderFactories()` for provider factories.
- `client.GetTestClient()` for destroy checks and sweepers.
- `utils.RandomWithPrefix()` or the package equivalent for test names.
- `envchecks.ComposePreChecks` with `envchecks.RedisCloudCheck` and every
  additional provider or environment check the scenario needs.

Drive HCL through `config.StaticFile` or `config.StaticDirectory`. Put the HCL in
`testdata/*.tf` and declare variables there. Pass values with
`config.Variables`; do not interpolate a Go string with `fmt.Sprintf`.

For every acceptance test touched by a migration, define `ConfigVariables`
inline in each `resource.TestStep`:

```go
{
    ConfigFile: config.StaticFile("testdata/example.tf"),
    ConfigVariables: config.Variables{
        "name": config.StringVariable(name),
    },
}
```

Do not assign the variables map before `Steps`, and do not hide it behind a
helper. Repeat the literal when two steps pass the same values. A reader should
see the configuration inputs beside the exact step that consumes them.

Treat a testdata file as a reusable configuration shape rather than a single
scenario. Reuse one parameterised file when only values differ; create another
file when the HCL structure differs.

Give every migrated data source its own acceptance test case in the new
Framework package. When the SDKv2 test combined resource and data-source
coverage for convenience or speed, split that coverage during the migration.
The data-source test may provision the resource it needs, but make assertions
only about the data source. This keeps each scenario focused and lighter, while
parallel test execution removes the speed advantage of combining them.

## Verify the migration

Run checks in increasing order of cost:

1. Format changed Go files and run `git diff --check`.
2. Run unit tests for the migrated package and every shared helper or custom-type
   package changed by the migration.
3. Run `go build ./...`, `go vet ./...`, and `make fmtcheck`.
4. Compare the resulting protocol-v5 schema with the SDKv2 schema, especially
   nested block kinds and Required, Optional, and Computed flags.
5. Run the relevant acceptance tests when credentials and a live test account
   are available. Report missing credentials as a skipped verification rather
   than treating them as a passing test.

For manual checks, build the provider and use the development override described
in `AGENTS.md`. Keep resource and data-source configurations in separate
`playground/` directories. Apply the resource first, then plan the data source so
its values are known during planning rather than deferred until apply.

Before handing off, inspect the complete branch diff against its actual parent.
Confirm that registration moved exactly once, no SDKv2 callers were stranded,
and unrelated schema or documentation changes did not enter the migration.
