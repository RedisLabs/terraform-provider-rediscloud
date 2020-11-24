---
layout: "rediscloud"
page_title: "Provider: Redis Enterprise Cloud"
description: |-
  Terraform provider RedisCloud.
---

# Redis Enterprise Cloud Provider

Use this paragraph to give a high-level overview of your provider, and any configuration it requires.

Use the navigation to the left to read about the available resources.

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
 
* `url` - (Optional) This is the URL of Redis Enterprise Cloud and will default to `https://api.redislabs.com/v1`.
This can also be set by the `REDISCLOUD_URL` environment variable. 

* `api_key` - (Optional) This is the Redis Enterprise Cloud API key. It must be provided but can also be set by the
`REDISCLOUD_ACCESS_KEY` environment variable.

* `secret_key` - (Optional) This is the Redis Enterprise Cloud API secret key. It must be provided but can also be set
by the `REDISCLOUD_SECRET_KEY` environment variable.
