---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_acl_role"
description: |-
  ACL Role resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_acl_rule

Creates a Role in your Redis Enterprise Cloud Account.

## Example Usage

```hcl
resource "rediscloud_acl_role" "role-resource-implicit" {
  name = "fast-admin"
  rule {
    # An implicit dependency is recommended
    name = rediscloud_acl_role.cache_reader.name
    # Implicit dependencies used throughout
    database {
      subscription = rediscloud_active_active_subscription_database.subscription-resource-1.id
      database     = rediscloud_active_active_subscription_database.database-resource-1.db_id
      regions      = [
        for r in rediscloud_active_active_subscription_database.database-resource-1.override_region : r.name
      ]
    }
    database {
      subscription = rediscloud_subscription.subscription-resource-2.id
      database     = rediscloud_subscription_database.database-resource-2.db_id
    }
  }
}

resource "rediscloud_acl_role" "role-resource-explicit" {
  name = "fast-admin"
  rule {
    name = "cache-reader"
    # Active-Active database omitted for brevity
    database {
      subscription = 123456
      database     = 9830
    }
  }
  # An explicit resource dependency can be used if preferred
  depends_on = [
    rediscloud_acl_rule.cache_reader,
    rediscloud_subscription.subscription-resource-2,
    rediscloud_subscription_database.database-resource-2
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A meaningful name for the role. Must be unique. **This can be modified, but since the Role is
  referred to
  by name (and not ID), this could break existing references. See the [User](rediscloud_acl_user.md) resource
  documentation.**
* `rule` - (Required, minimum 1) A set of rule association objects, documented below.

The `rule` block supports:

* `name` (Required) - Name of the Rule. It is recommended an implicit dependency is used here. `depends_on` could be
  used instead by waiting for a Rule resource with a matching `name`.
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
