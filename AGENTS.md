# Agent guide — terraform-provider-rediscloud

> **Status: living guide.** Working notes for agents on this repo — general dev
> workflow plus task-specific playbooks. It lives in the repo; improve it as you
> learn more (see [Maintaining this document](#maintaining-this-document)),
> ideally in the same PR as the work that taught you the thing.

---

## Project basics

`terraform-provider-rediscloud` is a Terraform provider written in Go. It is
mid-migration from the legacy `terraform-plugin-sdk/v2` to
`terraform-plugin-framework`, with both running side by side behind a
**protocol-v5 mux** — so most changes must "keep existing acceptance tests
green".

- **Build / format:** `make build` (compiles into `bin/` and writes
  `bin/developer_overrides.tfrc` for local runs) · `make fmtcheck` (goimports
  `-local github.com/RedisLabs/...`) · `go build ./...` · `go vet ./...`.
- **Test:** `go test ./...` for unit tests; `make testacc` (`TF_ACC=1`, real API
  creds — can't run without a live account) for acceptance tests. Current test
  patterns live in the migration playbook (§7–§8).
- **Run locally:** after `make build`, `export
  TF_CLI_CONFIG_FILE=$PWD/bin/developer_overrides.tfrc` so Terraform uses your
  built provider; put scratch configs in `playground/` (it's gitignored).
- **Key directories:** `provider/` (all provider code — one package per
  resource/data source), `docs/` (**published to the Terraform Registry** — no
  internal notes here), `playground/` (gitignored manual harnesses), `scripts/`,
  `bin/`.
- **Sandbox:** `make build`, `go build`/`vet`/`test`, and
  terraform-with-dev-overrides often need the sandbox disabled; the first time
  you hit it, confirm the user's per-session preference.

---

## Migration playbook (SDKv2 → Plugin Framework)

The provider is migrated one data source / resource at a time. Most rules below
fall out of the protocol-v5 mux and out of "keep existing acceptance tests
green".

**Reading guide:** §2 is split by kind and each subsection is self-contained —
read only the one you're migrating. Everything else applies to **both** kinds;
the few kind-specific bullets are tagged _(data sources)_ / _(resources)_.

### 0. Before you start

- **The old SDKv2 source is the baseline spec — but not bug-free.** Tickets are
  templated and vague; the behavioural spec is the existing SDKv2 code, so
  replicate what it did by default. Treat it as fallible, though: some quirks
  are genuine bugs, and the migration is a fair time to fix or improve things
  **as long as the change is non-breaking** (schema and observable behaviour
  stay backwards-compatible; see §4). When you spot a likely bug or a worthwhile
  improvement, **stop and confirm with the user** before deviating — don't
  silently "fix" it, and don't silently carry a known bug forward.
- **Migrations are often stacked.** A task may depend on helpers/files added on
  an earlier migration branch that are **not on `main`** (e.g.
  `provider/pro/flatten.go`). Verify prerequisites exist on your base
  before writing code (`git show HEAD:<path>`). If stacked, the **PR base should
  be the parent migration branch**, not `main`, so the diff stays scoped.

### 1. Schema type translation (non-negotiable mux v5 rules)

- **Nested structures are BLOCKS, never nested attributes.**
  `TypeList`/`TypeSet` \+ `Elem: &schema.Resource{}` becomes
  `schema.ListNestedBlock` / `schema.SetNestedBlock` — **not** the
  `*NestedAttribute` variants. Nested attributes require protocol v6 and will
  **panic at runtime** under the v5 mux. `TypeSet` → `SetNestedBlock`;
  `TypeList` → `ListNestedBlock`. See the comment in
  `provider/regions/datasource_regions.go`.
- **Primitive collections are ATTRIBUTES.** `TypeList`/`TypeSet` +
  `Elem: &schema.Schema{}` (list/set of primitives, e.g. strings) →
  `schema.ListAttribute`/`SetAttribute` with `ElementType`; `TypeMap` →
  `schema.MapAttribute`. Only list/set **of objects** are blocks.

### 2. Package & file layout

Each migrated data source / resource gets its own package dir, unless it belongs
to an existing grouping (e.g. `provider/activeactive`, `provider/pro`). After
migrating, delete the old SDKv2 implementation.

#### 2.1 Data sources

Blueprint: `provider/essentialsplan` (`rediscloud_essentials_plan`) — read it
with its PR/commit history. Three files:

- `datasource_<x>.go` — interface asserts (`var _ datasource.DataSource = ...`,
  `...WithConfigure`), `New<X>DataSource`, `Metadata`, `Configure`, `Schema`.
- `datasource_<x>_model.go` — the `tfsdk`-tagged model struct.
- `datasource_<x>_read.go` — `Read` + any flatten helpers.

#### 2.2 Resources

Blueprint: `provider/cloudaccount` (`rediscloud_cloud_account`) — read it with
its PR/commit history. Three files:

- `resource_<x>.go` — interface asserts (`resource.Resource`,
  `...WithConfigure`, `...WithImportState`), `New<X>Resource`, `Metadata`,
  `Configure`, `Schema`.
- `resource_<x>_model.go` — the `tfsdk`-tagged model struct.
- `resource_<x>_crud.go` — `Create`/`Read`/`Update`/`Delete`/`ImportState` +
  wait helpers.

**Resource CRUD gotchas** (not obvious from the framework docs):

- **Create/Update can't delegate to Read** the way SDKv2 did
  (`return ...Read(...)`) — the framework request/response signatures differ.
  Factor a `read<X>IntoModel(...) (notFound bool)` helper and call it from all
  three; on not-found, `resp.State.RemoveResource(ctx)` (the framework's
  `d.SetId("")`).
- **Fields the API never returns must be left UNTOUCHED by that helper** (e.g.
  secret key, console user/pass, sign-in URL). Assigning
  `StringPointerValue(nil)` silently wipes them to null — SDKv2 just never
  `d.Set` them, preserving the config/state value. Tests skip them via
  `ImportStateVerifyIgnore`.
- **State-change waiters stay on SDKv2.** A framework resource still imports
  `terraform-plugin-sdk/v2/helper/retry` (`retry.StateChangeConf` /
  `WaitForStateContext`) — there's no framework-native poller, so this is
  expected, not a leftover to "fix".
- **A resource that polls for a ready state must not orphan the remote object on
  poll failure.** When Create (or Update) provisions the object then waits for
  it to become active/ready, a timed-out or failed wait must still write the
  object to state before returning the error — persist the `id` and the last
  observed status via `resp.State.Set` — otherwise the created object leaks
  (untracked, unrecoverable, and the next apply makes a duplicate). SDKv2 got
  this for free (`d.SetId()` before the wait is persisted even on error); the
  framework does not, so do it explicitly. Have the poll helper _return_ the
  last observed status so you can record it without a second read (fall back to
  a null status if the first poll never landed). Precedent:
  `provider/cloudaccount` resource `Create`.
- **Delete is not idempotent, by design.** Redis TF resources assume their infra
  is managed **only** by Terraform, so `Delete` propagates the API error rather
  than swallowing a NotFound. If the object was removed out-of-band,
  `terraform destroy` **fails** — that's expected, not a bug to "fix" (don't add
  NotFound-tolerance to `Delete`). Contrast the first gotcha: `Read` _does_
  treat NotFound as "gone" and drops the resource from state; `Delete` does not.

### 3. Registration

- Add the constructor to `provider/framework_provider.go` (`DataSources()` /
  `Resources()`) and import the package.
- Remove the old entry from `provider/sdk_provider.go`, leaving a marker:
  `// Note: <public_name> is served by the Plugin Framework provider`.
- `Metadata` TypeName = `req.ProviderTypeName + "_<suffix>"`. **Watch
  public-name vs file-name mismatches**: `rediscloud_subscription` is the
  pro/flexible subscription but its package/type is "pro" — suffix is
  `_subscription`, not `_pro_subscription`.
- **The Framework has no implicit `id`.** SDKv2 had a mandatory implicit `id`;
  the Framework treats `id` as an ordinary attribute — nothing in the framework,
  the mux, or `terraform-plugin-testing` requires one. So whether to add an `id`
  depends on _why_ it would exist:
  - _Migrating from SDKv2:_ **add** an explicit `Computed` `id` `StringAttribute`
    (e.g. `strconv.Itoa(subId)`). The SDKv2 version always exposed an `id`, so
    keeping it is backward compatibility — existing configs/state/tests reference
    `.id`. This is the only reason the rule reads "always add an id" for migrations.
  - _Brand-new (no SDKv2 predecessor):_ add an `id` **only if it's functionally
    meaningful** (e.g. it's the object's real identifier and is useful to
    reference downstream); **skip it otherwise**. There's no compatibility
    constraint, so don't invent a placeholder id just to have one — an absent
    attribute is more honest than a sentinel that implies meaning it lacks.
  - Precedent: the net-new plural `rediscloud_subscriptions` data source
    (`provider/pro`) **omits** a top-level `id` — a list has no meaningful single
    identifier, and each element already carries its own `id`. Contrast the older
    SDKv2 aggregate data sources (`rediscloud_database_modules`,
    `rediscloud_subscription_peerings`), which set `id = "ALL"` as a sentinel
    purely because SDKv2 _required_ a non-empty id.

### 4. Schema & docs fidelity

- Preserve the schema **exactly**: attribute names, types,
  computed/optional/required, block nesting — otherwise acceptance tests break.
  Diff against the old schema; note per-field differences between siblings (e.g.
  pro-sub `name` is `Computed`+`Optional`, AA `name` is `Required`).
- Replace placeholder descriptions (`"Self-explanatory"`) with short, meaningful
  ones in the files you touch. Keep the schema description, the sibling
  resource/data source, and the `docs/` markdown consistently worded.
- Only touch `docs/` markdown if the schema actually changes — a faithful
  migration keeps it identical (exception: placeholder cleanup, when asked).

### 5. Value mapping (scalars)

- **null-for-nil** (see blueprint): `StringPointerValue` / `Float64PointerValue`
  / `BoolPointerValue` (nil → null); ints →
  `Int64Value(int64(redis.IntValue(...)))` (0 for nil).
- Fields the old code always stringifies (e.g. an int id) →
  `types.StringValue(strconv.Itoa(redis.IntValue(...)))` (always present, never
  null).
- Match SDKv2 special-case defaults, e.g. `public_endpoint_access` defaults to
  `true` when the API returns nil.
- **Empty-string vs null — testing gotcha.** `terraform-plugin-testing` treats
  an **absent** attribute as equal to `""`. A framework `types.StringNull()`
  renders as _absent_ in the test flatmap, satisfying **both**
  `TestCheckResourceAttr(k, "")` **and** `TestCheckNoResourceAttr(k)`. So
  null-for-nil is safe even when a test asserts `== ""`; you rarely need
  `StringValue("")`.
- **Prefer `rediscloud-go-api` constants over hardcoded literals.** Wherever the
  SDK exposes a value you'd otherwise hardcode — status strings, provider/enum
  values, etc. — reference the constant (`cloud_accounts.StatusActive`,
  `cloud_accounts.StatusDeleted`, `cloud_accounts.ProviderValues()`, …) rather
  than a magic string. It's a non-breaking readability/robustness win — the value
  tracks the SDK — that fits the migration, so make the swap when you spot one.
  Precedent: the status constants in `provider/cloudaccount`'s wait helpers and
  the `provider_type` `OneOf(cloud_accounts.ProviderValues()...)` validator.

### 6. Helpers

Two concerns: building framework values from API data (6.1), and reusing or
retiring the shared helper set (6.2).

#### 6.1 Flatten helpers (collections)

- Build nested collections with attr-type maps + `types.ObjectValue` +
  `types.ListValue`/`SetValue`; primitive collections via `types.ListValueFrom`
  / `MapValueFrom`. This is the `provider/pro` style (`FlattenMaintenance`,
  `FlattenPricing`).
- Alternative **typed-struct** style: `tfsdk`-tagged nested structs +
  `types.ObjectValueFrom` / `ListValueFrom` (the `provider/regions` style).
  Cleaner and type-checked, but mixes idioms if the file also uses the
  hand-rolled style.
- **Model fields stay `types.List` / `types.Set`.** The framework has no generic
  typed list value and the shared `pro.Flatten*` helpers return `types.List`,
  so receiving fields must match. Do **not** convert model fields to `[]struct`:
  you lose null/unknown fidelity, can't assign shared-helper output, and diverge
  from precedent.
- A flatten's behavioural spec is the **old SDKv2 flatten** — replicate its
  quirks (e.g. the pro cloud-details `isResource=false` path leaves region-level
  `networking_vpc_id` unset and always returns a non-nil `resource_tags` map).
- **Deterministic ordering.** The API can return nondeterministically-ordered
  lists (e.g. pricing) → sort by a composite key inside the flatten to kill a
  perpetual plan diff. Shared helpers already do this — inherit it, don't change
  it.

#### 6.2 Shared helpers & old-helper cleanup

- Shared subscription helpers live in the `provider/pro` package —
  `FilterSubscriptions` (in `datasource_rediscloud_pro_subscription.go`),
  `FlattenMaintenance` / `FlattenPricing` (in `flatten.go`), and the
  `CMK_ENABLED_STRING` constant. Reuse; don't duplicate.
- Filter-helper naming stays `filter<Singular>` (`pro.FilterSubscriptions` +
  `filterSubscription` predicate) — consistent with ~14 other data sources.
- **Grep the WHOLE provider for remaining callers before deleting any old
  helper.** SDKv2 code often still needs the map-based `Flatten*` helpers and
  constants (e.g. `CMK_ENABLED_STRING`); only remove what's now unused. Grep on
  the **correct (stacked) base** — a prior migration may already have removed
  callers.
- If a duplicated helper's only remaining caller is another SDKv2 file that can
  use the migrated equivalent (identical signature), **repoint it and delete the
  duplicate** rather than leaving a near-empty file. (We repointed the SDKv2
  AA-regions data source to the shared `pro.FilterSubscriptions` and deleted the
  duplicate helper.)
- **Leave a removal note on SDKv2-only helpers.** A helper kept alive only
  because SDKv2 code still calls it is migration debt — mark it in place so a
  later session deletes it instead of guessing, e.g.
  `// SDKv2-only: remove once the remaining callers (X, Y) are migrated to the framework.`
  Use a plain comment, not a godoc `// Deprecated:` marker (which would flag
  every caller in linters). When the grep above shows no SDKv2 callers remain,
  delete the helper and its note.

### 7. Tests

- Keep tests schema-compatible so they pass unchanged.
- **Unit-test the pure helpers you introduce/migrate.** Standalone helpers
  (filters, `Flatten*`, sort keys) get table-driven tests in
  `package <pkg>_test` with testify — decoupled from TF/mux, no acceptance
  creds. This is the cheap regression net the migration otherwise lacks (e.g. it
  pins the pricing deterministic sort). Exercise unexported helpers through
  their exported callers. Example: `provider/pro/flatten_test.go`.
- **Acceptance tests** need real API creds and can't run locally; the
  `terraform-plugin-testing` framework runs them only when `TF_ACC=1` (set by
  `make testacc`). There is no `EXECUTE_TESTS` or `testing.Short()` gate.
- **Prechecks go through the `envchecks` package.** Set
  `PreCheck: envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, <more>...)`
  — it runs every check and fails **once** with the full list of missing env
  vars (the old per-package `BasicPreCheck` and the `utils.AccRequiresEnvVar`
  **skip** are gone; there's no skip-vs-fail split any more — everything fails).
  Ready-made checks: `RedisCloudCheck` (URL + access/secret key),
  `AWSProviderCheck`, `GCPProviderCheck`, or `RequireEnvVars(t, ...)` for ad-hoc
  vars. When a test needs a var's **value**, use a value+check pair so the same
  var is read _and_ gated — `name, check := envchecks.AWSBYOCValueAndCheck()`
  (also `ValueAndCheck(key)`, `GCPProjectValueAndCheck`,
  `AwsPeeringValueAndCheck`) — and pass `check` into `ComposePreChecks`.
- **Unchanged wiring:**
  `ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories()`; get an API
  client for `CheckDestroy` / sweepers via `client.GetTestClient()`; name
  resources with `utils.RandomWithPrefix()` (honours `TEST_RESOURCE_PREFIX`).
- **A `testdata/*.tf` file maps to a config _form_, not a scenario.** Scenarios
  that share structure and differ only in values reuse **one** parameterised
  file via per-step `ConfigVariables`. Add a separate `.tf` only when the
  _structure_ differs (e.g. a GCP cloud account taking different arguments than
  AWS) — don't fork just to hardcode different values.
- _(Data sources)_ Relocate a data-source test only if it's a **standalone**
  test file → move it into the package as `package <pkg>_test`. If the
  assertions are **embedded** in a combined resource+datasource test, leave them
  (AA precedent). The essentialsplan blueprint relocated its standalone test.

### 8. Verify

- `go build ./...` · `go vet ./...` · `make fmtcheck` (goimports
  `-local github.com/RedisLabs/...`).
- Run unit tests for the package + `provider/utils`.
- A pre-existing acceptance test that hard-fails on a missing env var is
  probably **environmental and not your regression** — confirm it fails
  identically on the base.
- Prepare testing .tf files for the user to perform manual testing
  - drive the freshly-built provider from `playground/` (gitignored; split into
    `datasource/` and `resource/` if needed)
  - **The shared test env is wiped daily**. This applies both to resources and
    data sources, but has more impact on data sources, because usually have no
    object to read. In that case first `apply` a resource in
    `playground/resource/` to create one, then point the data-source config in
    `playground/datasource/` at it and `plan`.
  - **Setup:** `make build`, then export
    `TF_CLI_CONFIG_FILE=$PWD/bin/developer_overrides.tfrc` and
    `REDISCLOUD_ACCESS_KEY` / `REDISCLOUD_SECRET_KEY` (+ any vars the resource
    needs); dev-overrides let you skip `terraform init`.
  - _(Data sources)_ - Keep the data-source `.tf` in its own dir, separate from
    any resource `.tf`.\*\* A data source relies on infra that already exists,
    so sharing a config with the resource that creates that same infra, forces
    data source fields to be calculated as "known after apply" and hides the
    values until you run a second `tf apply`. When data source and resource are
    kept separate, `terraform apply` on the resource first, allows a
    `terraform plan` on the data source, ran as a second step, to read the data
    at plan time with full values.

---

## Maintaining this document

Living doc — update it when you finish a migration or hit something surprising:

- **Add durable, reusable knowledge only** — recurring decisions, non-obvious
  mux behaviours, precedents, gotchas. Not one-off task details or anything
  obvious from the code / git history.
- **Correct/merge in place; don't append contradictions.** If a rule turns out
  wrong or narrower/broader than stated, fix it where it lives.
- **Cite a concrete anchor** (a precedent file, blueprint, verified test
  behaviour) so the next session can check it rather than trust it blindly.
- **Keep §2.1 and §2.2 self-contained as both paths grow.** Data sources and
  resources are both exercised now; when one gains a gotcha, check whether the
  other needs the mirror note — but don't make one subsection depend on the
  other.
- When you rely on a rule, **sanity-check it still holds** (files move, helpers
  migrate). If it names a file/helper that no longer exists, update the anchor.
