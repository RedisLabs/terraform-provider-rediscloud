---
page_title: "Redis Cloud: rediscloud_langcache_api_key"
description: |-
  LangCache data-plane API key resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_langcache_api_key

Creates a named data-plane API key for a Redis LangCache. The generated secret is returned only during creation and is stored in Terraform state as a sensitive value.

## Example Usage

```hcl
resource "rediscloud_langcache_api_key" "application" {
  cache_id = rediscloud_langcache.example.id
  name     = "application"
}

output "langcache_api_key" {
  value     = rediscloud_langcache_api_key.application.api_key
  sensitive = true
}
```

## Argument Reference

* `cache_id` - (Required) The LangCache ID.
* `name` - (Required) The name of the API key.

Changing either argument replaces the API key.

## Attribute Reference

* `id` - The API key ID.
* `api_key` - The generated data-plane API key. This value is sensitive and is returned only during creation.
* `obfuscated_token` - The obfuscated API key returned by the Admin API.
* `created_at` - The API key creation timestamp.

## Import

Import using the LangCache ID and API key ID separated by `/`. The secret `api_key` value cannot be recovered during import.

```shell
terraform import rediscloud_langcache_api_key.application cache-id/api-key-id
```
