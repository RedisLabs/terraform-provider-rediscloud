---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_acl_rule"
description: |-
  ACL Rule resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_acl_rule

Creates a Rule (a.k.a Redis Rule, Redis ACL) in your Redis Enterprise Cloud Account.

## Example Usage

```hcl
resource "rediscloud_acl_rule" "rule-resource" {
  name = "my-rule"
  rule = "+@read ~cache:*"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A meaningful name for the rule. Must be unique.
* `rule` - (Required) The ACL rule itself, built up as permissions/restrictions written in
  the [ACL Syntax](https://docs.redis.com/latest/rc/security/access-control/data-access-control/configure-acls/#define-permissions-with-acl-syntax).

### Timeouts

The `timeouts` block allows you to
specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 5 mins) Used when creating the Rule.
* `update` - (Defaults to 5 mins) Used when updating the Rule.
* `delete` - (Defaults to 5 mins) Used when destroying the Rule.

## Attribute reference

* `id` - Identifier of the Rule created.
* `name` - The Rule's name.
* `rule` - The ACL Rule itself.

## Import

`rediscloud_acl_rule` can be imported using the Identifier of the Rule, e.g.

```
$ terraform import rediscloud_acl_rule.rule-resource 123456
```
