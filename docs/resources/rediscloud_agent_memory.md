---
page_title: "Redis Cloud: rediscloud_agent_memory"
description: |-
  Agent Memory resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_agent_memory

Creates a Redis Agent Memory service backed by an existing Redis Cloud database.
The resource manages the Agent Memory service configuration, not the memory
entries stored through the Agent Memory data-plane API.

## Prerequisites

The backing database must meet the current
[Agent Memory service prerequisites](https://redis.io/docs/latest/operate/iris/agent-memory/create-service/#prerequisites-and-limitations).
During public preview, its `default` user must be enabled. Redis Flex databases,
databases using AWS PrivateLink, and Active-Active databases are not supported.
Check the linked service documentation for changes to these limitations before
selecting a backing database.

## Example Usage

### Minimal service

```hcl
resource "rediscloud_agent_memory" "example" {
  name        = "example-memory"
  database_id = rediscloud_subscription_database.example.db_id
}
```

When optional time-based memory settings are omitted, Redis Cloud applies the
service defaults and the provider stores the values returned by the API.

### Service with memory settings

```hcl
resource "rediscloud_agent_memory" "example" {
  name        = "example-memory"
  database_id = rediscloud_subscription_database.example.db_id

  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_strategy        = "INSTRUCT"
  extraction_cadence_seconds = 300
}
```

### Service with customer-managed models

```hcl
resource "rediscloud_agent_memory" "example" {
  name        = "example-memory"
  database_id = rediscloud_subscription_database.example.db_id

  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_strategy        = "INSTRUCT"
  extraction_cadence_seconds = 300

  llm {
    provider = "openai"
    model    = "gpt-4o-mini"

    credentials {
      type    = "apiKey"
      api_key = var.agent_memory_llm_api_key
      api_key_version = 1
    }
  }

  embedding {
    provider = "openai"
    model    = "text-embedding-3-small"

    credentials {
      type    = "apiKey"
      api_key = var.agent_memory_embedding_api_key
      api_key_version = 1
    }
  }
}
```

### Model credentials

Model `credentials.api_key` is sensitive and is stored in Terraform plan and
state, like other credentials in this provider. Sensitive values are hidden in
normal CLI output; that does not encrypt saved artifacts. Protect access to the
state backend and saved plans. BYOK does not require Terraform 1.11.1.

```hcl
variable "llm_api_key" {
  type      = string
  sensitive = true
}

# Inside llm.credentials; the same fields are available in embedding.credentials.
credentials {
  type            = "apiKey"
  api_key         = var.llm_api_key
  api_key_version = 1
}
```

Each configured key requires an `api_key_version` greater than zero. Changing
`api_key` updates the credential in place. Increment its version to resubmit the
same key, for example after restoring provider-side access. LLM and embedding
credentials have independent versions and may reference the same key or
different keys. TTL-only updates do not resend unchanged credentials.

Redis Cloud does not return model credentials. Refresh preserves the values
already held in state, while import cannot recover keys or their Terraform-only
versions. The first apply with configured credentials after import sends the
keys and records them in state without replacing the store.

Configurations from earlier development builds may declare model input variables
with `ephemeral = true`. Remove that setting before using sensitive model
credentials. Ephemeral values cannot be assigned to an argument stored in state.

### Service with advanced memory configuration

```hcl
resource "rediscloud_agent_memory" "example" {
  name        = "example-memory"
  database_id = rediscloud_subscription_database.example.db_id

  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_strategy        = "INSTRUCT"
  extraction_cadence_seconds = 300

  summarization {
    enabled          = true
    trigger_strategy = "event_count"

    event_count {
      threshold    = 20
      retain_count = 10
    }
  }

  custom_memory_types {
    name        = "preference"
    description = "User preferences captured from conversations"

    fields {
      name        = "topic"
      type        = "str"
      description = "Preference topic"
    }

    extraction_strategy {
      enabled = true
      prompt  = "Extract durable user preferences."
    }
  }

  long_term_memory_exclusions {
    enabled = true

    built_in_detectors {
      enabled = true

      detectors {
        id      = "email"
        enabled = true
        action  = "redact"
      }
    }

    custom_detectors {
      enabled = true

      detectors {
        name    = "account_number"
        enabled = true
        action  = "redact"

        matcher {
          kind = "regex"

          regex {
            pattern = "ACCT-[0-9]+"
          }
        }
      }
    }

    semantic {
      enabled = true
      prompt  = "Do not remember secrets, recovery codes, or payment credentials."
    }
  }
}
```

## Argument Reference

The following arguments are required:

* `name` - The name of the Agent Memory service.
  Must contain at least one non-whitespace character. Maximum length is validated by the Agent Memory API.
* `database_id` - The ID of the Redis Cloud database that backs the Agent Memory service. Changing it requires replacement.
  Must be an integer. Validity and access to the database are checked by the Agent Memory API.

The following arguments are optional:

* `short_term_ttl_seconds` - Short-term memory TTL in seconds.
  The API accepts 1 second to 1 year.
* `long_term_ttl_seconds` - Long-term memory TTL in seconds.
  The API accepts 1 second to 1 year. This must be configured explicitly when `embedding` is configured.
* `extraction_strategy` - How long-term memories are extracted from sessions.
  Accepted values are validated by the Agent Memory API. Changing an explicitly configured strategy requires replacement because the API does not support updating it in place.
* `extraction_cadence_seconds` - How often the extraction pipeline runs while a session is active, in seconds. Accepted values are validated by the Agent Memory API.
* `llm` - Customer-managed LLM configuration. If configured, `embedding` must also be configured. Both `provider` and `model` must be explicitly configured.
  * `provider` - Model provider. The Agent Memory API validates this against the providers supported by the service. Changing an explicitly configured provider requires replacement.
  * `model` - Model name advertised by the provider.
  * `credentials` - Required when creating a store with customer-managed models; optional on existing/imported stores. When present, `type`, `api_key`, and `api_key_version` must be configured.
    * `type` - Credential type validated by the Agent Memory API.
    * `api_key` - Provider API key stored as a sensitive value in plan and state. Requires `api_key_version`.
    * `api_key_version` - Positive credential revision. Increment to resend the same key; changing the key also triggers an in-place update.
* `embedding` - Customer-managed embedding model configuration for long-term memory. If configured, `llm` must also be configured. Both `provider` and `model` must be explicitly configured.
  * `provider` - Model provider. The Agent Memory API validates this against the providers supported by the service. Changing an explicitly configured provider requires replacement.
  * `model` - Model name advertised by the provider. Changing an explicitly configured model requires replacement.
  * `credentials` - Required when creating a store with customer-managed models; optional on existing/imported stores. When present, `type`, `api_key`, and `api_key_version` must be configured.
    * `type` - Credential type validated by the Agent Memory API.
    * `api_key` - Provider API key stored as a sensitive value in plan and state. Requires `api_key_version`.
    * `api_key_version` - Positive credential revision. Increment to resend the same key; changing the key also triggers an in-place update.
* `summarization` - Automatic session summarization configuration.
  * `enabled` - Whether automatic summarization is enabled.
  * `trigger_strategy` - Summarization trigger strategy. Accepted values are validated by the Agent Memory API; when omitted, the provider records the value returned by the API. Explicit empty strings are rejected.
  * `event_count` - Event-count thresholds. Numeric limits are validated by the Agent Memory API. Both fields must be configured when this block is present.
    * `threshold` - Number of messages that triggers summarization. Must be greater than `retain_count`.
    * `retain_count` - Number of most recent messages retained in full.
* `custom_memory_types` - Custom long-term memory types. Quantity limits are validated by the Agent Memory API.
  * `name` - Non-empty custom type name. Format and maximum length are validated by the Agent Memory API.
  * `description` - Non-empty custom type description. Maximum length is validated by the Agent Memory API.
  * `fields` - Structured fields for this custom memory type.
    * `name` - Non-empty field name. Format and maximum length are validated by the Agent Memory API.
    * `type` - Non-empty field type validated by the Agent Memory API. Examples include `str`, `int`, `float`, `bool`, `list[str]`, `list[float]`, and `object`.
    * `description` - Non-empty field description. Maximum length is validated by the Agent Memory API.
  * `extraction_strategy` - Optional extraction strategy for this custom memory type.
    * `enabled` - Whether extraction is enabled for this custom type.
    * `prompt` - Non-empty prompt used to extract memories of this custom type. Required when this extraction strategy block is present. Additional limits are determined by the Agent Memory service.
* `long_term_memory_exclusions` - Exclusion policy applied to newly extracted long-term memories.
  * `enabled` - Master toggle for long-term memory exclusions.
  * `built_in_detectors` - Built-in sensitive-data detector configuration.
    * `enabled` - Whether built-in detectors are enabled.
    * `detectors` - Built-in detector selections.
      * `id` - Non-blank built-in detector ID validated by the Agent Memory API. Examples include `credit-card`, `email`, `ip-address`, `phone`, and `us-ssn`; the available catalogue is managed by the service.
      * `enabled` - Whether this detector is enabled.
      * `action` - Action when the detector matches. Examples include `redact` and `drop`; accepted values are validated by the Agent Memory API. When omitted, the provider records the value returned by the API. Explicit empty strings are rejected.
  * `custom_detectors` - Custom regex detector configuration.
    * `enabled` - Whether custom detectors are enabled.
    * `detectors` - Custom detector definitions.
      * `name` - Non-empty custom detector name. Format, maximum length and conflicts with built-in detector names are validated by the Agent Memory API.
      * `enabled` - Whether this detector is enabled.
      * `action` - Action when the detector matches. Accepted values are validated by the Agent Memory API; when omitted, the provider records the value returned by the API. Explicit empty strings are rejected.
      * `matcher` - Detector matcher configuration.
        * `kind` - Non-empty matcher kind validated by the Agent Memory API. When `kind` is `regex`, the `regex` block is required. The published API currently documents `regex`; other kinds needing additional fields require SDK/schema support.
        * `regex.pattern` - Non-empty regex pattern used by the matcher. Maximum length and syntax are validated by the Agent Memory API. Patterns known to match empty text are rejected locally.
  * `semantic` - Semantic exclusion configuration.
    * `enabled` - Whether semantic exclusions are enabled.
    * `prompt` - Prompt describing concepts that should not be kept in long-term memory.

Exclusions apply when the service extracts new long-term memories from sessions.
They do not filter session events or summaries, or long-term memories written
directly by the application. Changes do not re-evaluate existing memories.
Semantic exclusions guide the extraction model but do not guarantee exclusion.
See the [service exclusion documentation](https://redis.io/docs/latest/operate/iris/agent-memory/create-service/#sensitive-data-exclusions)
for scope and limitations.

Removing every `detectors` block from a configured detector group sends an empty
list to clear its selections. Disabling a group preserves explicitly configured
definitions so it can be enabled again later. Removing the optional group itself
does not send a deletion request for its stored definitions.

All time-based arguments are expressed in seconds.

Most configuration can be updated in place. Changes to `database_id` or an
explicitly configured `extraction_strategy` require store replacement because the
API only accepts these settings on create. Omitting `extraction_strategy` keeps
the existing strategy; it does not reset it to the service default. A configured
value that remains unknown during planning can conservatively require replacement,
even if it later resolves to the previous value.

For customer-managed models, `llm` and `embedding` must be configured together
on create. Adding or removing both blocks requires replacement. Changing
`llm.provider`, `embedding.provider`, or `embedding.model` also requires
replacement. `llm.model` can be updated within the existing provider, and model
credentials can be rotated without replacement by updating `credentials.api_key`
or incrementing `credentials.api_key_version` to resend the same key.

A replacement destroys and recreates the store with a new store ID. It does not
migrate the existing agent context to that new identity. Use
`lifecycle.prevent_destroy` when destruction must be blocked.

Deleting a store, including during replacement, is destructive. The
[Agent Memory service documentation](https://redis.io/docs/latest/operate/iris/agent-memory/view-service/#delete-service)
states that deletion permanently removes the associated memory data, service
configuration, search index, and stored model provider key, and revokes the
service API keys. The replacement creates a new store from the Terraform
configuration; it does not migrate existing memories or preserve the old
service API keys.

If an immutable selector depends on a value that is unknown during planning,
Terraform may conservatively plan replacement even when that value later
resolves to the previous value. Model blocks must explicitly configure their provider and model selectors.
Explicitly unknown expressions remain unknown.

A replacement plan is not a validation dry run. If replacement inputs resolve to
invalid values only during apply, the old store may already have been destroyed
before the error is reported. Use values known during planning or
`lifecycle.prevent_destroy` instead of relying on late validation to preserve it.

The control-plane API stores model credentials, while data-plane operations
such as long-term memory create/search validate them when the model provider is
called. If a credential is missing or invalid, those data-plane operations can
return an `Invalid Model Credentials` error.

For `custom_memory_types`, the provider follows the Agent Memory API behavior:
new custom memory types can be added in place, and `extraction_strategy.prompt`
or `extraction_strategy.enabled` can be updated for existing custom memory
types. Removing an existing custom memory type, changing its description, or
changing its fields, or removing its extraction strategy requires replacement of
the entire Agent Memory service because the API cannot apply those changes in place.
To disable extraction without replacement, keep the strategy and set `enabled = false`.
Reordering types or fields does not require replacement. Unknown existing names or
immutable definitions can conservatively require replacement, even if they later
resolve to their previous values; unknown values belonging only to new types do
not make existing definitions immutable changes.

## Attribute Reference

* `id` - The Agent Memory store ID.
* `endpoint` - The primary data-plane API endpoint for the Agent Memory service.
* `endpoints` - Regional data-plane API endpoints for the Agent Memory service.
  * `url` - Endpoint URL.
  * `provider` - Cloud provider hosting the endpoint.
  * `region` - Cloud region of the endpoint.
  * `egress_ips` - Egress IP addresses used by the endpoint.
  * `is_accessible` - Whether the endpoint is reachable from the store's database network.
* `status` - The current provisioning status.
* `created_at` - The Agent Memory service creation timestamp.

## Timeouts

An optional `timeouts` block configures operation deadlines using duration strings
such as `30s`, `5m`, or `1h`.

| Operation | Default |
| --- | --- |
| `create` | `30m` |
| `read` | `5m` |
| `update` | `30m` |
| `delete` | `30m` |

```hcl
timeouts {
  create = "45m"
  read   = "10m"
  update = "45m"
  delete = "45m"
}
```

Place this block inside the Agent Memory resource. Each deadline covers the whole
operation, including task polling and any follow-up read. An earlier caller
deadline or cancellation still applies. Import uses a fixed `5m` deadline because
resource configuration is not available during import.

A timeout does not undo an operation already accepted by Redis Cloud. If creation
may have succeeded without returning an ID, reconcile the remote store before
retrying to avoid duplicates.

If creation returns a store ID but the immediate follow-up read fails or returns
invalid metadata, the provider saves that ID and the known configuration and
reports a warning. It does not adopt invalid metadata or return a creation error
that would taint the acknowledged store. Investigate or refresh the existing
store before retrying. Subsequent read and update failures remain errors; a
persistent invalid API response can therefore block further operations.

## Import

`rediscloud_agent_memory` can be imported using its store ID:

```shell
$ terraform import rediscloud_agent_memory.example store-id
```

For imported services with customer-managed models, the provider imports the
model metadata returned by Redis Cloud, such as `llm.provider`, `llm.model`,
`embedding.provider`, and `embedding.model`. Model provider credentials are not
returned by the Agent Memory API, so they are not populated during import.
If the Terraform configuration includes `credentials` blocks after import, the
first plan can show an in-place update to send those credentials
back to Redis Cloud. This is expected and does not replace the Agent Memory
service. To rotate credentials after import, update the relevant
`credentials.api_key` values, then run `terraform apply`.
