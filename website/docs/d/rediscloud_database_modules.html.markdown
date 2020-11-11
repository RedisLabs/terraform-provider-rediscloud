---
layout: "rediscloud"
page_title: "RedisCloud: rediscloud_database_modules"
sidebar_current: "docs-rediscloud-database-modules"
description: |-
  Database Modules data source in the Terraform provider RedisCloud.
---

# rediscloud_database_module

Use this data source to get a list of supported database moodules.  A single database module can be applied to a database.

## Example Usage

```hcl
data "rediscloud_database_modules" "example" {
}
```

## Attributes Reference

`modules` A list of database modules.

Each module entry provides the following attributes

* `name` The identifier assigned by the database module

* `description` A meaningful description of the database module
