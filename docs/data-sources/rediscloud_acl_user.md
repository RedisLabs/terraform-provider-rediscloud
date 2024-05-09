---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_acl_user"
description: |-
  ACL User data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_acl_user

The User data source allows access to an existing Rule within your Redis Enterprise Cloud Account.

## Example Usage

```hcl
data "rediscloud_acl_user" "example" {
  name = "fast-admin-john"
}

output "rediscloud_acl_user" {
  value = data.rediscloud_acl_user.example.id
}
```

## Argument Reference

* `name` - (Required) The name of the User to filter returned subscriptions

## Attribute reference

* `id` - Identifier of the found User.
* `name` - The User's name.
* `role` - The name of the User's Role.
