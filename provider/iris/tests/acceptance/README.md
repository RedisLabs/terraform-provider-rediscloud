# Iris acceptance tests

Each test provisions its own Essentials subscription and database through
Terraform, then creates uniquely named `${TEST_RESOURCE_PREFIX:-tf-test}-iris-acc-*`
stores. Destroy removes the Iris resources before the database and subscription.
Pre-existing databases and stores are never deleted. Store cleanup uses `flush=false`.
The shared infrastructure selects the free AWS `30MB` plan in `us-east-1` with
Redis Stack and the default Redis version. Override `TF_VAR_region` or
`TF_VAR_redis_version` when the test environment requires different values.
Run serially in an account without an existing Essentials subscription, as in
the repository's other Essentials acceptance tests. The standard precheck rejects
an occupied account before apply. Fixture assembly copies only the checked-in
database and scenario files, so local Terraform override files are not loaded.
For a paid run, set `TF_VAR_plan_name` to an available plan and
`TF_VAR_payment_method_id` to an existing account payment method.

Configure credentials through environment variables; do not put them in fixtures:

```sh
export REDISCLOUD_IRIS_AGENT_MEMORY_URL=https://your-memory-control-plane.example.com
export REDISCLOUD_IRIS_IDENTITY_URL=https://your-identity-control-plane.example.com
# Set REDISCLOUD_URL, REDISCLOUD_ACCESS_KEY and REDISCLOUD_SECRET_KEY securely.
TF_ACC=1 nix develop --command go test ./provider/iris/tests/acceptance \
  -run '^TestAccResourceIris' -parallel=1 -count=1 -v -timeout=45m
```

These tests are not part of the repository's CI workflows. Run them manually
against a configured Iris deployment and a disposable Redis Cloud account.

Coverage follows the CAPI store scenarios: create and readiness, rename and TTL
update, an empty follow-up plan, rejected configuration, import, refresh after
out-of-band deletion, immutable custom-field replacement, and confirmed destroy.
A second-database replacement is not run because each fixture provisions one
database. Store-scoped CAPI API-key and sweeper tests are not reused:
those exercise separate resources and API paths. The shared sweeper registers
Iris cleanup separately and limits store deletion to old stores in test-named
databases inside old test-named Essentials subscriptions. The resource prefix and
age checks apply as well. The Identity key sweeper deletes a key only when its
name carries the test prefix, it is old enough, and every grant points at a store
the store sweeper owns; without the Agent Memory URL it deletes nothing. Zero-age
Iris sweeping requires `IRIS_SWEEP_ALLOW_ACTIVE=1`; use that only after tests stop
in an isolated account. `make sweep` continues infrastructure cleanup after a
dependent sweeper fails and still reports the failures. Missing Iris URL skips
the live tests.

The standalone memory-type test uses three sibling types in one temporary store
to exercise concurrent registry writes. It covers
create, extraction-prompt updates, an empty plan, import, plan-time rejection of
immutable description/field edits, deletion with extraction still enabled, and
the service's refusal to reuse the retired name. The type is removed while the
store remains, so deleting the parent cannot hide a failed unregister.

Run both store and standalone-type scenarios:

```sh
TF_ACC=1 nix develop --command go test ./provider/iris/tests/acceptance \
  -run '^TestAccResourceIrisAgentMemory' -count=1 -v -timeout=45m
```

Customer-model creation and import use one shared `AGENT_MEMORY_MODEL_API_KEY`
for both LLM and embedding calls, matching the UI. The legacy pair
`AGENT_MEMORY_LLM_API_KEY` and `AGENT_MEMORY_EMBEDDING_API_KEY` is also accepted
when the shared variable is unset. Without model keys, that case is skipped.
Model keys are supplied through the environment, never checked into fixtures.

The fixture uses native attributes and `custom_memory_type` / `field` blocks.
Model keys reach write-only `llm_api_key` and `embedding_api_key` attributes through
Terraform environment variables. The provider supplies the API credential type
`API_KEY`; no model JSON is needed. Import verification ignores unrecoverable
inline custom-type snapshots and credential-version metadata.

## Identity Service

Set `REDISCLOUD_IRIS_IDENTITY_URL` to the service's base URL, including any
gateway prefix. Authenticate with `REDISCLOUD_IRIS_IDENTITY_TOKEN`, or use
`REDISCLOUD_ACCESS_KEY` and `REDISCLOUD_SECRET_KEY` when the Cloud gateway
supports those headers. An explicit bearer token takes precedence.

```sh
export REDISCLOUD_IRIS_IDENTITY_URL=https://cloud.example.com/iris/identity
TF_ACC=1 nix develop --command go test ./provider/iris/tests/acceptance \
  -run '^TestAccResourceIrisIdentity' -count=1 -v -timeout=45m
```

The API-key test also requires the Agent Memory settings above. It creates two
temporary stores and a key, then checks rename, adding/removing a grant, empty
plans, expiry reset and deletion. API-key import is not supported. The key tenant
is omitted by default to test service stamping; set `REDISCLOUD_IRIS_IDENTITY_TENANT` if the installation requires an
explicit tenant. Key tokens are never printed by the test checks. Terraform
retains them in its temporary state as documented by the resource.

Missing Identity URL skips the API-key test. Other configuration and API errors
fail the test.

The shared sweeper also registers `rediscloud_iris_identity_api_key` cleanup for
interrupted runs. With the Identity URL configured, it lists Agent Memory keys
and deletes only keys with the test prefix older than `SWEEP_AGE_THRESHOLD`
(default 24 hours). It completes pagination before deletion and runs before
store cleanup. Ordinary test cleanup still deletes only that test's keys.
