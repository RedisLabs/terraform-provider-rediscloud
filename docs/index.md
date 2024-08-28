---
layout: "rediscloud"
page_title: "Provider: Redis Enterprise Cloud"
description: |-
   The Redis Enterprise Cloud provider is used to interact with the resources supported by Redis Enterprise Cloud. The provider needs to be configured with the proper credentials before it can be used..
---

# Redis Enterprise Cloud Provider

The Redis Enterprise Cloud provider is used to interact with the resources supported by Redis Enterprise Cloud . The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available provider resources and data sources.

## Configure Redis Enterprise Cloud Programmatic Access

In order to setup authentication with the Redis Enterprise Cloud provider a programmatic API key must be generated for Redis Enterprise Cloud. The [Redis Enterprise Cloud documentation](https://redis.io/docs/latest/integrate/terraform-provider-for-redis-cloud/) contains the most up-to-date instructions for creating and managing your key(s) and IP access.

## Example Usage

```hcl
provider "rediscloud" {
}

# Example resource configuration
resource "rediscloud_subscription" "example" {
  # ...
}
```

## Argument Reference

In addition to [generic `provider` arguments](https://www.terraform.io/docs/configuration/providers.html)
(e.g. `alias` and `version`), the following arguments are supported in the Redis Cloud
`provider` block:

* `api_key` - (Optional) This is the Redis Enterprise Cloud API key. It must be provided but can also be set by the
`REDISCLOUD_ACCESS_KEY` environment variable.

* `secret_key` - (Optional) This is the Redis Enterprise Cloud API secret key. It must be provided but can also be set
by the `REDISCLOUD_SECRET_KEY` environment variable.
