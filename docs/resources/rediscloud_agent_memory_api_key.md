---
page_title: "Redis Cloud: rediscloud_agent_memory_api_key"
description: |-
  Agent Memory data-plane API key resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_agent_memory_api_key

Creates a named data-plane API key for a Redis Agent Memory service.

The generated secret is returned only during creation and is stored in Terraform
state as a sensitive value. Sensitive values are hidden from normal Terraform
CLI output, but they are still stored in state and should be protected with an
appropriate Terraform backend.

## Example Usage

```hcl
resource "rediscloud_agent_memory" "example" {
  name        = "example-memory"
  database_id = rediscloud_subscription_database.example.db_id
}

resource "rediscloud_agent_memory_api_key" "application" {
  store_id = rediscloud_agent_memory.example.id
  name     = "application"
}

output "agent_memory_api_key" {
  value     = rediscloud_agent_memory_api_key.application.api_key
  sensitive = true
}
```

## Argument Reference

The following arguments are supported:

* `store_id` - (Required) The Agent Memory store ID. Must contain only letters, numbers, and hyphens. The Agent Memory API validates the referenced store.
* `name` - (Required) The name of the API key. Must be non-empty and must not contain whitespace. Maximum length is validated by the Agent Memory API.

Changing either argument replaces the API key.

## Attribute Reference

* `id` - The API key ID.
* `api_key` - The generated data-plane API key. This value is sensitive and is returned only during creation.
* `obfuscated_token` - The obfuscated API key returned by the Admin API.
* `created_at` - The API key creation timestamp.

## Timeouts

An optional `timeouts` block configures operation deadlines using duration strings
such as `30s`, `5m`, or `1h`. Create, read, and delete each default to `5m`.

```hcl
timeouts {
  create = "10m"
  read   = "5m"
  delete = "10m"
}
```

Place this block inside the API key resource. Changing only timeout values keeps
the existing key ID and secret; it does not rotate the key. Key import performs no
network request, so it has no operation timeout.

An earlier caller deadline or cancellation still applies. A timeout does not undo
an operation already accepted by Redis Cloud. If key creation may have succeeded,
verify and reconcile the remote keys before retrying.

## Import

Import using the Agent Memory store ID and API key ID separated by `/`. The secret `api_key` value cannot be recovered during import.

```shell
$ terraform import rediscloud_agent_memory_api_key.application store-id/api-key-id
```
