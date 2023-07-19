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
resource "rediscloud_acl_user" "user-resource-implicit" {
  name     = "fast-admin-john"
  # An implicit dependency is recommended
  role     = rediscloud_acl_role.fast_admin.name
  password = "mY.passw0rd"
}

resource "rediscloud_acl_user" "user-resource-explicit" {
  name     = "fast-admin-john"
  role     = "fast-admin"
  password = "mY.passw0rd"

  # An explicit resource dependency can be used if preferred
  depends_on = [
    rediscloud_acl_role.fast_admin
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, change forces recreation) A meaningful name for the User. Must be unique. An error occurs if a
  user tries to connect to
  a `memcached` database with the username `admin`.
* `role` - (Required) The name of the Role held by the User. It is recommended an implicit dependency is used
  here. `depends_on` could be used instead by waiting for a Role resource with a matching `name`.
* `password` - (Required, change forces recreation) The password for this ACL User. Must contain a lower-case letter, a
  upper-case letter, a
  number and a special character. Can be updated but since it is not returned by the API, we have no way of detecting
  drift, so the entity would be entirely replaced. Take special care with multiple versions of Terraform State.

### Timeouts

The `timeouts` block allows you to
specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 5 mins) Used when creating the User.
* `update` - (Defaults to 5 mins) Used when updating the User.
* `delete` - (Defaults to 5 mins) Used when destroying the User.

## Attribute reference

* `id` - Identifier of the User created.
* `name` - The User's name.
* `role` - The User's role name.

## Import

`rediscloud_acl_user` can be imported using the Identifier of the User, e.g.

```
$ terraform import rediscloud_acl_user.user-resource 123456
```
