# Iris Agent Memory acceptance tests

These tests make real API calls. They use the existing database `51785871` and
create uniquely named `tf-iris-acc-*` stores. The database and pre-existing stores
are never deleted. Store cleanup uses `flush=false`.

Configure credentials through environment variables; do not put them in fixtures:

```sh
export REDISCLOUD_IRIS_AGENT_MEMORY_URL=https://app-qa.qa.redislabs.com
# Set REDISCLOUD_ACCESS_KEY and REDISCLOUD_SECRET_KEY securely.
TF_ACC=1 nix develop --command go test ./provider/iris/tests/acceptance \
  -run '^TestAccResourceIrisAgentMemory_' -count=1 -v -timeout=45m
```

Coverage follows the CAPI store scenarios: create and readiness, rename and TTL
update, an empty follow-up plan, rejected configuration, import, refresh after
out-of-band deletion, immutable custom-field replacement, and confirmed destroy.
A second-database replacement is not run because this fixture uses the single
supplied database. Store-scoped CAPI API-key and sweeper tests are not reused:
those exercise separate resources and API paths.

Customer-model creation and import require `AGENT_MEMORY_LLM_API_KEY` and
`AGENT_MEMORY_EMBEDDING_API_KEY`; otherwise that case is explicitly skipped.
Model keys are supplied through the environment, never checked into fixtures.

Live app-qa verification on 2026-11-20 passed CRUDI and custom-field replacement.
The customer-model case was skipped because model-provider keys were not supplied.
The final account check found no remaining test stores and retained the original
`gosho` store. This is scenario parity for the store lifecycle, not a claim that
the CAPI-only API-key/sweeper or second-database scenarios were executed.
