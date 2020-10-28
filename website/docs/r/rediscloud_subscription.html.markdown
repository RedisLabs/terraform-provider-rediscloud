---
layout: "rediscloud"
page_title: "RedisCloud: rediscloud_subscription"
sidebar_current: "docs-rediscloud-subscription"
description: |-
  Subscription resource in the Terraform provider RedisCloud.
---

# rediscloud_subscription

Subscription resource in the Terraform provider RedisCloud.

## Example Usage

```hcl
resource "rediscloud_subscription" "example" {
  sample_attribute = "foo"
}
```

## Argument Reference

The following arguments are supported:

* `sample_attribute` - Sample attribute.
