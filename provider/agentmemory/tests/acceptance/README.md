# Agent Memory acceptance tests

These tests run Terraform against the real Redis Cloud API. Each test creates its
own Pro subscription, databases, Agent Memory store and API key, then destroys
them. The account and region must support Agent Memory. Fixtures use AWS,
replication, TLS and an enabled default database user.

The suite covers:

- Store creation, in-place updates, import and an unchanged plan.
- API key creation, import without recovering its secret, and replacement.
- API rejection of an invalid TTL without changing the existing store.
- Real sweeper cleanup and recovery after a store is deleted outside Terraform.
- Store and dependent key replacement when the database or custom field type changes.
- Customer-managed models and credential version updates, when model keys are supplied.
- API verification of tracked IDs, metadata and deletion, including superseded resources.

## Run

Set `REDISCLOUD_URL`, `REDISCLOUD_ACCESS_KEY` and `REDISCLOUD_SECRET_KEY` for a test
account. Managed-model cases use the repository's default Terraform CLI; the
customer-model case requires Terraform 1.11.1 or later. From the repository root:

```sh
make testacc-check TEST_PATTERN='^TestAccResourceRedisCloudAgentMemory_'
TEST_RESOURCE_PREFIX=tf-am-local-unique TF_LOG= \
  make testacc TEST_PATTERN='^TestAccResourceRedisCloudAgentMemory_'
```

`go test ./...` skips the live tests unless `TF_ACC` is set. It still runs the
sweeper ownership checks without cloud credentials.
The live cases run serially because credential inputs and the cleanup rehearsal
use process environment variables.

Optional settings:

| Variable | Default or purpose |
| --- | --- |
| `AGENT_MEMORY_TEST_REGION` | `us-east-1` |
| `AGENT_MEMORY_TEST_CARD_TYPE` | `Visa` |
| `AGENT_MEMORY_TEST_CARD_LAST_FOUR` | `5556`, the card used by existing acceptance fixtures |
| `AWS_TEST_CLOUD_ACCOUNT_NAME` | Use a named AWS BYOC account instead of Redis-managed hosting |
| `TEAM_RESOURCE_TAG`, `OWNER_RESOURCE_TAG` | Required when using the BYOC account |
| `AGENT_MEMORY_LLM_API_KEY`, `AGENT_MEMORY_EMBEDDING_API_KEY` | Enable the customer-managed model case |
| `AGENT_MEMORY_MODEL` | `gpt-4o-mini` |
| `AGENT_MEMORY_EMBEDDING_MODEL` | `text-embedding-3-small` |

The customer-managed model case uses ephemeral sensitive input variables and
checks that model secrets are absent from plan and state. Its version update
resubmits the same valid keys; it does not claim to test rotation to different
provider credentials. It skips when either key is missing or the selected CLI
is below 1.11.1. The managed fixture contains no model credential variables.

To select the minimum supported CLI for the BYOK case, supply the two model keys
and run:

```sh
TF_ACC_TERRAFORM_VERSION=1.11.1 TEST_RESOURCE_PREFIX=tf-am-byok-unique TF_LOG= \
  make testacc TEST_PATTERN='^TestAccResourceRedisCloudAgentMemory_CustomerModels$'
```

Unset `TF_ACC_TERRAFORM_PATH` when selecting a downloadable version; an explicit
binary path takes precedence.

The invalid-TTL case checks HTTP 400 and reads the existing TTL back before
another apply. Validation responses can contain only a generic `Bad Request`,
without the rejected field or constraint; the test does not assume those details
are available.

## Cleanup after interruption

Use the same unique `TEST_RESOURCE_PREFIX` as the interrupted run. First remove
Agent Memory stores, then let the existing subscription sweeper remove their
databases and subscriptions:

```sh
export TEST_RESOURCE_PREFIX=tf-am-local-unique
export SWEEP_AGE_THRESHOLD=0s
go test ./provider/agentmemory/tests/acceptance -run='^$' \
  -sweep=ALL -sweep-run=rediscloud_agent_memory -timeout=20m
go test ./provider -run='^$' \
  -sweep=ALL -sweep-run=rediscloud_subscription -timeout=30m
```

The Agent Memory sweeper requires both the store name and its subscription to
belong to these fixtures. It skips stores with missing IDs or creation times.
Without `SWEEP_AGE_THRESHOLD`, it only selects stores at least 24 hours old.

## CI

The existing PR and main workflows call the ephemeral-account workflow in a
separate Agent Memory job, using their existing CI and acceptance gates. Its opt-in
`sweep_agent_memory` input removes fixture stores before the existing subscription
cleanup. That input defaults to false for all other callers.

The new jobs only provision fixtures when Agent Memory files, SDK dependency
files or the shared acceptance workflow change, or during the main workflow's
manual run. Other resource-only PRs skip the Agent Memory jobs.

The shared workflow does not export model-provider secrets, so its default CI run
covers Redis-managed models. Run the optional customer-managed model case locally
with the variables above.
