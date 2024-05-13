---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_acl_role"
description: |-
  ACL Role resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_acl_role

Creates a Role in your Redis Enterprise Cloud Account.

## Example Usage

```hcl
resource "rediscloud_acl_rule" "rule-resource" {
  name = "my-rule"
  rule = "+@read ~cache:*"
}

resource "rediscloud_acl_role" "role-resource" {
  name = "my-role"
  rule {
    name = rediscloud_acl_rule.rule-resource.name
    database {
      subscription = rediscloud_flexible_subscription.subscription-resource.id
      database     = rediscloud_flexible_database.database-resource.db_id
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A meaningful name for the role. Must be unique.
* `rule` - (Required, minimum 1) A set of rule association objects, documented below.

The `rule` block supports:

* `name` (Required) - Name of the Rule.
* `database` - (Required, minimum 1) a set of database association objects, documented below.

The `database` block supports:

* `subscription` (Required) - ID of the subscription containing the database.
* `database` (Required) - ID of the database to which the Rule should apply.
* `regions` (Optional) - For databases in Active/Active subscriptions only, the regions to which the Rule should apply.

### Timeouts

The `timeouts` block allows you to
specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 5 mins) Used when creating the Role.
* `update` - (Defaults to 5 mins) Used when updating the Role.
* `delete` - (Defaults to 5 mins) Used when destroying the Role.

## Attribute reference

* `id` - Identifier of the Role created.
* `name` - The Role's name.
* `rule` - The Rules associated with the Role.

The `rule` block supports:

* `name` - Name of the Rule.
* `database` - The Databases the Rule applies to.

The `database` block supports:

* `subscription` ID of the subscription containing the database.
* `database` ID of the database to which the Rule should apply.
* `regions` The regions to which the Rule should apply, if appropriate to the database.

## Import

`rediscloud_acl_role` can be imported using the Identifier of the Role, e.g.

```
$ terraform import rediscloud_acl_role.role-resource 123456
```
