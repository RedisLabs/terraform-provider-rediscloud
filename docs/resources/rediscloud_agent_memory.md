---
page_title: "Redis Cloud: rediscloud_agent_memory"
description: |-
  Agent Memory resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_agent_memory

Creates a Redis Agent Memory service backed by an existing Redis Cloud database.
The resource manages the Agent Memory service configuration, not the memory
entries stored through the Agent Memory data-plane API.

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
    }
  }

  embedding {
    provider = "openai"
    model    = "text-embedding-3-small"

    credentials {
      type    = "apiKey"
      api_key = var.agent_memory_embedding_api_key
    }
  }
}
```

Use sensitive variables for model provider credentials. `llm.credentials.api_key`
and `embedding.credentials.api_key` can reference different keys, or the same
key if the model provider credential is allowed to call both the LLM and
embedding models. Redis Cloud stores these credentials as write-only values.

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
  Must be 1-64 characters and contain at least one non-whitespace character.
* `database_id` - The ID of the Redis Cloud database that backs the Agent Memory service.
  Must be greater than zero.

The following arguments are optional:

* `short_term_ttl_seconds` - Short-term memory TTL in seconds.
  The API accepts 1 second to 1 year.
* `long_term_ttl_seconds` - Long-term memory TTL in seconds.
  The API accepts 1 second to 1 year. This must be configured explicitly when `embedding` is configured.
* `extraction_strategy` - How long-term memories are extracted from sessions.
  Accepted values are validated by the Agent Memory API. Changing an existing store strategy is blocked during planning because the API does not support updating it in place.
* `extraction_cadence_seconds` - How often the extraction pipeline runs while a session is active, in seconds. Accepted values are validated by the Agent Memory API.
* `llm` - Customer-managed LLM configuration. If configured, `embedding` must also be configured.
  * `provider` - Model provider. The Agent Memory API validates this against the providers supported by the service.
  * `model` - Model name advertised by the provider.
  * `credentials` - Provider credential configuration.
    * `type` - Credential type validated by the Agent Memory API.
    * `api_key` - Provider API key. This value is sensitive and write-only in the Agent Memory API.
* `embedding` - Customer-managed embedding model configuration for long-term memory. If configured, `llm` must also be configured.
  * `provider` - Model provider. The Agent Memory API validates this against the providers supported by the service.
  * `model` - Model name advertised by the provider.
  * `credentials` - Provider credential configuration.
    * `type` - Credential type validated by the Agent Memory API.
    * `api_key` - Provider API key. This value is sensitive and write-only in the Agent Memory API.
* `summarization` - Automatic session summarization configuration.
  * `enabled` - Whether automatic summarization is enabled.
  * `trigger_strategy` - Summarization trigger strategy. Accepted values are validated by the Agent Memory API; when omitted, the provider records the value returned by the API. Explicit empty strings are rejected.
  * `event_count` - Event-count thresholds. Numeric limits are validated by the Agent Memory API. Both fields must be configured when this block is present.
    * `threshold` - Number of messages that triggers summarization. Must be greater than `retain_count`.
    * `retain_count` - Number of most recent messages retained in full.
* `custom_memory_types` - Custom long-term memory types. Quantity limits are validated by the Agent Memory API.
  * `name` - Non-empty custom type name. Format and maximum length are validated by the Agent Memory API.
  * `description` - Custom type description.
  * `fields` - Structured fields for this custom memory type.
    * `name` - Non-empty field name. Format and maximum length are validated by the Agent Memory API.
    * `type` - Field type. Supported values are `str`, `int`, `float`, `bool`, `list[str]`, `list[float]`, and `object`.
    * `description` - Field description.
  * `extraction_strategy` - Optional extraction strategy for this custom memory type.
    * `enabled` - Whether extraction is enabled for this custom type.
    * `prompt` - Prompt used to extract memories of this custom type.
* `long_term_memory_exclusions` - Policy defining content that must not be kept in long-term memory.
  * `enabled` - Master toggle for long-term memory exclusions.
  * `built_in_detectors` - Built-in sensitive-data detector configuration.
    * `enabled` - Whether built-in detectors are enabled.
    * `detectors` - Built-in detector selections.
      * `id` - Built-in detector ID. Supported values are `credit-card`, `email`, `ip-address`, `phone`, and `us-ssn`.
      * `enabled` - Whether this detector is enabled.
      * `action` - Action when the detector matches. Accepted values are validated by the Agent Memory API; when omitted, the provider records the value returned by the API. Explicit empty strings are rejected.
  * `custom_detectors` - Custom regex detector configuration.
    * `enabled` - Whether custom detectors are enabled.
    * `detectors` - Custom detector definitions.
      * `name` - Non-empty custom detector name. Format and maximum length are validated by the Agent Memory API.
      * `enabled` - Whether this detector is enabled.
      * `action` - Action when the detector matches. Accepted values are validated by the Agent Memory API; when omitted, the provider records the value returned by the API. Explicit empty strings are rejected.
      * `matcher` - Detector matcher configuration.
        * `kind` - Non-empty matcher kind validated by the Agent Memory API. When `kind` is `regex`, the `regex` block is required. The current SDK/schema can represent only `kind` and the optional `regex` block; other kinds requiring additional fields need SDK/schema support.
        * `regex.pattern` - Regex pattern used by the matcher.
  * `semantic` - Semantic exclusion configuration.
    * `enabled` - Whether semantic exclusions are enabled.
    * `prompt` - Prompt describing concepts that should not be kept in long-term memory.

All time-based arguments are expressed in seconds.

Most configuration can be updated in place. Changes to `database_id` or an
explicitly configured `extraction_strategy` are blocked during planning. The
provider does not automatically replace the store for these changes: a new store
has a different identity and may no longer reference the existing agent context.
Plan a migration and explicitly manage the old and new stores when needed.
Explicit removal, destroy, or Terraform replacement commands can still delete a
store; this validation is not a general deletion safeguard.

For customer-managed models, `llm` and `embedding` must be configured together
when the Agent Memory service is first created. The Agent Memory API does not
allow a store created with platform-managed models to move to customer-managed
models later. The provider blocks that plan instead of sending a request that
the API will reject. After creation, `llm.provider`, `embedding.provider`, and
`embedding.model` are immutable. `llm.model` can be updated within the existing
provider, and model credentials can be rotated by updating `credentials.api_key`.
Credentials are write-only and cannot be recovered from the API during import.
The control-plane API stores model credentials, while data-plane operations
such as long-term memory create/search validate them when the model provider is
called. If a credential is missing or invalid, those data-plane operations can
return an `Invalid Model Credentials` error.

For `custom_memory_types`, the provider follows the Agent Memory API behavior:
new custom memory types can be added in place, and `extraction_strategy.prompt`
or `extraction_strategy.enabled` can be updated for existing custom memory
types. Removing an existing custom memory type, changing its description, or
changing its fields is blocked during planning because the API does not support
those operations in place. Terraform does not automatically replace the Agent
Memory service for those changes.

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
first plan can show an in-place update to send those write-only credentials
back to Redis Cloud. This is expected and does not replace the Agent Memory
service. To rotate credentials after import, update the relevant
`credentials.api_key` values and run `terraform apply`.
