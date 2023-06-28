---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_acl_user"
description: |-
  ACL User resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_acl_user

Creates a User in your Redis Enterprise Cloud Account.

## Example Usage

```hcl
resource "rediscloud_acl_user" "user-resource" {
  name = "fast-admin-john"
  role = "fast-admin"
  password = "mY.passw0rd"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A meaningful name for the User. Must be unique. An error occurs if a user tries to connect to a `memcached` database with the username `admin`.
* `role` - (Required) The name of the Role held by the User
* `password` - (Required) The password for this ACL User. Must contain a lower-case letter, a upper-case letter, a number and a special character. Can be updated but is not returned as an attribute.

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 3 mins) Used when creating the User.
* `update` - (Defaults to 3 mins) Used when updating the User.
* `delete` - (Defaults to 1 mins) Used when destroying the User.

## Attribute reference

* `id` - Identifier of the User created.
* `name` - The User's name.
* `role` - The User's role name.

## Import
`rediscloud_acl_user` can be imported using the Identifier of the User, e.g.

```
$ terraform import rediscloud_acl_user.user-resource 123456
```
