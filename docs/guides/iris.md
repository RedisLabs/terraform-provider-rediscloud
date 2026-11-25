---
page_title: "Direct Iris control planes"
description: |-
  Configure and manage Agent Memory and Identity Service through their direct APIs.
---

# Direct Iris control planes

The `rediscloud_iris_*` resources call the Agent Memory and Identity Service
control planes directly. Configure each service with its own base URL and bearer
token. The base URL can include a reverse-proxy path prefix; do not append an
operation path such as `/v1/api-keys`.

```hcl
provider "rediscloud" {
  iris_agent_memory_url = "https://memory-control.example.com"
  iris_identity_url     = "https://identity.example.com"
}
```

Supply tokens using `REDISCLOUD_IRIS_AGENT_MEMORY_TOKEN` and
`REDISCLOUD_IRIS_IDENTITY_TOKEN`, or the sensitive provider attributes
`iris_agent_memory_token` and `iris_identity_token`. URLs can also be supplied
using `REDISCLOUD_IRIS_AGENT_MEMORY_URL` and `REDISCLOUD_IRIS_IDENTITY_URL`.
Only configure the services you use. Use a service bearer token, or use the
Cloud API credentials described below for gateways that support them.

Iris-only configurations using bearer authentication do not require Redis Cloud
`api_key` and `secret_key`.
Supply those credentials as usual when also managing Redis Cloud resources.
The existing `rediscloud_agent_memory` and `rediscloud_agent_memory_api_key`
resources continue to use Redis Cloud CAPI and its credentials.

## Resources

- `rediscloud_iris_agent_memory` manages a memory store backed by an existing database.
- `rediscloud_iris_agent_memory_type` manages one custom type within a store.
- `rediscloud_iris_identity_api_key` manages a generic Identity Service API key and its grants.

Use the matching resource documentation for configuration examples. Agent Memory
stores and memory types support import; Identity API keys do not.
Use `.id` to reference a resource. The `store_id` input on a memory-type resource
identifies its parent store.
Avoid managing the same remote object through both an Iris and a CAPI resource.

## Native Terraform configuration

Settings use typed Terraform attributes and blocks. No resource accepts JSON
configuration. Simple settings such as TTLs are scalar attributes; repeated
memory types, fields, detectors and API-key grants are nested blocks.

```hcl
resource "rediscloud_iris_agent_memory" "example" {
  name                         = "application-memory"
  database_id                  = "your-existing-database-id"
  short_memory_ttl_seconds     = 86400
  long_term_memory_ttl_seconds = 31536000

  custom_memory_type {
    name               = "preference"
    description        = "Durable user preferences"
    extraction_prompt  = "Extract preferences explicitly stated by the user."
    extraction_enabled = true
    field {
      name        = "topic"
      description = "Preference topic"
      type        = "STRING"
    }
  }
}
```

Omitted scalar settings use the service defaults or retain the current service
value on update. Repeated blocks describe the complete collection; removing a
detector block removes that detector. Collection order does not affect plans.
`endpoints` is a native computed set of objects.
On existing stores, update summarization counts, detector selections or semantic
prompts without repeating unchanged group flags and sibling settings.
On creation, configure the parent enable flags explicitly when supplying those
settings or detector blocks. Setting a flag to `false` deliberately keeps that
group inactive; omitting the flag does not imply enabling it.

Inline `custom_memory_type` blocks are an initial creation snapshot. The store
read API omits them, so the provider retains them from configuration and cannot
recover them on import. Use separate `rediscloud_iris_agent_memory_type`
resources for ongoing type CRUD and imports. When importing a store, omit inline
custom types and import those types as separate resources instead.

When creating a store, select the LLM and embedding providers and model names
together. Classification requires its own provider and model name. For example, with
customer-supplied OpenAI credentials:

```hcl
variable "model_api_key" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "rediscloud_iris_agent_memory" "customer_models" {
  name                      = "customer-models"
  database_id               = "your-existing-database-id"
  llm_provider              = "openai"
  llm_model                 = "gpt-4o-mini"
  llm_api_key               = var.model_api_key
  llm_api_key_version       = 1
  embedding_provider        = "openai"
  embedding_model           = "text-embedding-3-small"
  embedding_api_key         = var.model_api_key
  embedding_api_key_version = 1
}
```

The provider sets the Iris credential discriminator to `API_KEY`; no credential
JSON or discriminator is needed in HCL. Optional classification settings use
`classification_provider`, `classification_model`, `classification_api_key` and
`classification_api_key_version`. Each model selection uses its own provider's
credential requirements. For providers configured by your deployment to need no
credentials, omit both the API key and its version. Iris validates provider and
model availability and whether credentials are required.
Model API keys are write-only sensitive attributes, omitted from plan and state.
Set a positive matching `*_api_key_version` and increment it whenever the key
changes. Credential versions trigger rotation; changing only a write-only value
does not produce a Terraform diff.
Supported same-provider model updates retain the stored credentials; unchanged
provider names, unrelated model settings and API keys need not be repeated.

Store database changes require replacement. Changes to an embedding model or
a model provider, or switching a store from platform models to its own LLM or
embedding model, also require replacement. Credential rotation and supported
same-provider LLM model changes use an update. Parent IDs and other create-only
fields require replacement.

Memory stores wait for `READY` after creation and update. A failed readiness
check retains the identifier and latest observed status in state. Operations
default to a 30-minute deadline; ordinary refresh defaults to five minutes.
All three Iris resources support a native `timeouts` block to override these
deadlines, for example:

```hcl
timeouts {
  create = "45m"
  read   = "2m"
  update = "45m"
  delete = "45m"
}
```

Changing only timeouts updates local Terraform state without an API mutation.
Ordinary refresh reads use the configured `read` timeout. During readiness
polling, individual GET attempts have a one-minute timeout, limited by the
remaining operation deadline. Readiness retries transient read failures;
writes retain the operation deadline and are never retried automatically.

If Iris cannot provision a new store, it can keep the store with the status
`UNAVAILABLE` and still return an error that omits the store identifier. Terraform
then holds no state for that store and does not track it. A retry creates another
store, so delete the unavailable one through Iris.

Standalone memory types reject description and field changes within the same
store and name. Removal permanently retires that name, including case variants.
If the follow-up read after type creation fails, Terraform retains the creation
response and reports a warning so a transient read failure cannot trigger replacement.
Use a distinct, unused name for a different schema. Destroy first disables
extraction, then deletes using the refreshed revision. External document
extractors must also be removed before the service permits deletion.
An existing type can change `extraction_enabled` without configuring its prompt.
Configure extraction when creating a type; Iris cannot add a strategy to an
existing type that was created without one.

Memory-type writes within the same store are serialized by the provider. Writes
include the revision last read from the service. An external conflicting write
produces an error for the next refresh and plan to resolve. The provider does not
retry with a newer revision automatically.

The Identity API key `token` is sensitive and returned only at creation. It is
retained through refresh and update. Identity API keys cannot be imported because
the service cannot return an existing token. Agent Memory imports cannot recover
model credentials or credential versions; configure those separately after import.

Iris deletes expired key records, so refresh removes an expired key from state
and Terraform plans to create it again. If `expires_at` is a fixed timestamp that
has already passed, that create fails at apply because Iris rejects past
timestamps; planning the rest of the configuration is unaffected. Change
`expires_at` to a future timestamp, or `0` for a non-expiring key, before
recreating it. Changing the expiry of an existing key, or replacing it by
changing `user_id`, rejects a known past timestamp at plan time, so Iris never
rejects the expiry of a replacement after the old key is deleted.
Terraform never silently extends a key's lifetime.

Changing `tenant` or `user_id` replaces the key, and Terraform deletes the
current key before it creates the new one. Iris rejects the new key unless every
store in its grants belongs to the new tenant, and the provider cannot check
that, so change the tenant together with its grants. Set
`lifecycle { create_before_destroy = true }` on the resource so that a rejected
replacement leaves the current key in place.

```hcl
resource "rediscloud_iris_identity_api_key" "application" {
  name   = "application-runtime"
  tenant = "tenant-b"

  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = rediscloud_iris_agent_memory.tenant_b.id
    actions       = ["read", "write"]
  }

  lifecycle {
    create_before_destroy = true
  }
}
```

This moves the key to `tenant-b` and grants it a store that belongs to that
tenant. Terraform creates the new key first and deletes the old one afterwards, so
if Iris rejects the new key the old key stays in place. The new key has a new
token; switch consumers to it, because the old token stops working once the old
key is deleted.

## Replacing Identity API keys

Use a second key resource to issue a new token:

1. Copy the existing resource under a new Terraform label and name, preserving
   its grants, tenant and `user_id` where configured. Apply with both keys present.
2. Switch consumers to the new resource's `token`, deploy and verify them.
3. Remove the old key resource and apply to revoke it.

Both keys remain usable during the handover, subject to their configured expiry.
Each resource creates an independent key with a new subject identity.

## Store deletion

`flush_on_delete` defaults to `false`, retaining short-term sessions when the
store is removed. Set it to `true` to request deletion of those sessions.
Long-term memories are retained by the service in either case. A resource
already missing from the service is removed from Terraform state.

## Redis Cloud gateway authentication

For Agent Memory behind a Cloud gateway, set `iris_agent_memory_url` to the
origin (for example, `https://memory-control.example.com`). For Identity behind
a Cloud gateway, set `iris_identity_url` to its base URL including any proxy
prefix (for example, `https://cloud.example.com/iris/identity`). Supply the
provider's `api_key` and `secret_key`, or `REDISCLOUD_ACCESS_KEY` and
`REDISCLOUD_SECRET_KEY`.
When no bearer token is supplied for a service, the provider uses
`x-api-key` and `x-api-secret-key` headers. An explicit bearer token takes
precedence. The gateway must support Cloud API authentication for that service.
An Identity Service reached directly authenticates with a bearer token issued by
its operator, set as `iris_identity_token`. Creating an API key does not issue
that token.
