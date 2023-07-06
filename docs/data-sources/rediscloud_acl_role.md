---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_acl_role"
description: |-
  ACL Role data source in the Terraform provider Redis Cloud.
---

# Data Source: rediscloud_acl_role

The Role data source allows access to an existing Role within your Redis Enterprise Cloud Account.

## Example Usage

```hcl
data "rediscloud_acl_role" "example" {
  name = "fast-admin"
}

output "rediscloud_acl_role" {
  value = data.rediscloud_acl_role.example.id
}
```

## Argument Reference

* `name` - (Required) The name of the Role to filter returned subscriptions

## Attribute reference

* `id` - Identifier of the found Role.
* `name` - The Role's name.
* `rules` - The Rules associated with the Role.

The `rules` list is made of objects with:

* `name` - Name of the Rule.
* `databases` - a list of database association objects, documented below.

The `databases` list is made of objects with:

* `subscription` ID of the subscription containing the database.
* `database` ID of the database to which the Rule should apply.
* `regions` The regions to which the Rule should apply, if appropriate to the database.
