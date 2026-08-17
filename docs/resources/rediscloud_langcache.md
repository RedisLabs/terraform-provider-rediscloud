---
page_title: "Redis Cloud: rediscloud_langcache"
description: |-
  LangCache resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_langcache

Creates a Redis LangCache backed by an existing Redis Cloud database.

## Example Usage

```hcl
resource "rediscloud_langcache" "example" {
  name                       = "example-cache"
  database_id                = rediscloud_subscription_database.example.db_id
  embedding_model_provider   = "openai"
  embedding_model_name       = "text-embedding-3-small"
  embedding_model_api_key    = var.openai_api_key
  default_search_threshold   = 0.1
  default_ttl_millis         = 3600000
  attributes                 = ["tenant", "language"]
  search_strategies          = ["exact", "semantic"]
}
```

## Argument Reference

The following arguments are required:

* `name` - The name of the LangCache.
* `database_id` - The ID of the Redis Cloud database that backs the LangCache.
* `embedding_model_provider` - The embedding provider ID.
* `embedding_model_name` - The embedding model ID.
* `default_search_threshold` - Default distance threshold used when searching cache entries.
* `default_ttl_millis` - Default entry TTL in milliseconds. `0` means entries are not stored; `-1` stores entries indefinitely.
* `attributes` - Custom attributes available for filtering cache entries. Use `[]` when no attributes are needed.

The following arguments are optional:

* `embedding_model_api_key` - API key for embedding providers that require one. This value is sensitive.
* `custom_model_base_url` - Base URL for a custom embedding model. Must be set together with `custom_model_dimensions`.
* `custom_model_dimensions` - Vector dimensions for a custom embedding model. Must be set together with `custom_model_base_url`.
* `search_strategies` - Search strategies in priority order. Supported values are `exact` and `semantic`.
* `flush_on_destroy` - Whether to flush cache entries when destroying the LangCache. Defaults to `false`.

All configuration can be updated in place except `database_id`, which replaces the LangCache.

## Attribute Reference

* `id` - The LangCache ID.
* `endpoint` - The API endpoint for the LangCache.
* `status` - The current provisioning status.

## Import

`rediscloud_langcache` can be imported using its cache ID:

```shell
terraform import rediscloud_langcache.example cache-id
```
