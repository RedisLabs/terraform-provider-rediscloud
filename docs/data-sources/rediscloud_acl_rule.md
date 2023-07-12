---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_acl_rule"
description: |-
  ACL Rule data source in the Terraform provider Redis Cloud.
---

# Data Source: rediscloud_acl_rule

The Rule (a.k.a Redis Rule, Redis ACL) data source allows access to an existing Rule within your Redis Enterprise Cloud Account.

## Example Usage

```hcl
data "rediscloud_acl_rule" "example" {
  name = "cache-reader-rule"
}

output "rediscloud_acl_rule" {
  value = data.rediscloud_acl_rule.example.id
}
```

## Argument Reference

* `name` - (Required) The name of the Rule to filter returned subscriptions

## Attribute reference

* `id` - Identifier of the found Rule.
* `name` - The Rule's name.
* `rule` - The ACL Rule itself.
