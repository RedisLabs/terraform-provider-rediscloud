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
resource "rediscloud_acl_rule" "rule-resource" {
  name = "cache-reader-rule"
  rules = [
    {
      name = "cache-reader-rule"
      databases = [
        {
          subscription = 123456
          database = 9829
          regions = ["us-east-1", "us-east-2"]
        }
      ]
    }
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A meaningful name for the role. Must be unique. **This can be modified, but since the Role is referred to
  by name (and not ID), this could break existing references. See User documentation.**
* `rules` - (Required) A list of rule association objects, documented below.

The `rules` list supports:

* `name` (Required) - Name of the Rule.
* `databases` - (Required) a list of database association objects, documented below.

The `databases` list supports:

* `subscription` (Required) - ID of the subscription containing the database.
* `database` (Required) - ID of the database to which the Rule should apply.
* `regions` (Optional) - For databases in Active/Active subscriptions only, the regions to which the Rule should apply.


### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 3 mins) Used when creating the rule.
* `update` - (Defaults to 3 mins) Used when updating the rule.
* `delete` - (Defaults to 1 mins) Used when destroying the rule.

## Attribute reference

* `id` - Identifier of the Role created.
* `name` - The Role's name.
* `rules` - The Rules associated with the Role.

The `rules` list is made of objects with:

* `name` - Name of the Rule.
* `databases` - a list of database association objects, documented below.

The `databases` list is made of objects with:

* `subscription` ID of the subscription containing the database.
* `database` ID of the database to which the Rule should apply.
* `regions` The regions to which the Rule should apply, if appropriate to the database.

## Import
`rediscloud_acl_role` can be imported using the Identifier of the Role, e.g.

```
$ terraform import rediscloud_acl_role.role-resource 123456
```
