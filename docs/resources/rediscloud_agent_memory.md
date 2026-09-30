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
  extraction_cadence_seconds = 300
}
```

### Service with advanced memory configuration

```hcl
resource "rediscloud_agent_memory" "example" {
  name        = "example-memory"
  database_id = rediscloud_subscription_database.example.db_id

  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
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
  The API accepts 1 second to 1 year.
* `extraction_cadence_seconds` - How often the extraction pipeline runs while a session is active. When configured, the API currently accepts 60-600 seconds.
* `summarization` - Automatic session summarization configuration.
  * `enabled` - Whether automatic summarization is enabled.
  * `trigger_strategy` - Summarization trigger strategy. Currently only `event_count` is supported.
  * `event_count` - Event-count thresholds.
    * `threshold` - Number of messages that triggers summarization. Must be greater than `retain_count`.
    * `retain_count` - Number of most recent messages retained in full.
* `custom_memory_types` - Custom long-term memory types. A store can define up to 3 custom memory types.
  * `name` - Custom type name. Must start with a letter and contain only letters, numbers, underscores, and hyphens.
  * `description` - Custom type description.
  * `fields` - Structured fields for this custom memory type.
    * `name` - Field name.
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
      * `action` - Action when the detector matches. Supported values are `redact` and `drop`.
  * `custom_detectors` - Custom regex detector configuration.
    * `enabled` - Whether custom detectors are enabled.
    * `detectors` - Custom detector definitions.
      * `name` - Custom detector name.
      * `enabled` - Whether this detector is enabled.
      * `action` - Action when the detector matches. Supported values are `redact` and `drop`.
      * `matcher` - Detector matcher configuration.
        * `kind` - Matcher kind. Currently only `regex` is supported.
        * `regex.pattern` - Regex pattern used by the matcher.
  * `semantic` - Semantic exclusion configuration.
    * `enabled` - Whether semantic exclusions are enabled.
    * `prompt` - Prompt describing concepts that should not be kept in long-term memory.

All time-based arguments are expressed in seconds.

Most configuration can be updated in place. Changing `database_id` or
`custom_memory_types` replaces the Agent Memory service. Custom memory type
replacement is conservative because the API does not support full in-place
redefinition of custom type names, descriptions, and fields.

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

## Import

`rediscloud_agent_memory` can be imported using its store ID:

```shell
$ terraform import rediscloud_agent_memory.example store-id
```
