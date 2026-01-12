---
page_title: "Redis Cloud: rediscloud_database_modules"
description: |-
  Database Modules data source in the Redis Cloud Terraform provider.
---


# Data Source: rediscloud_database_modules

The database modules data source allows access to a list of supported [Redis Enterprise Cloud modules](https://redislabs.com/redis-enterprise/modules).  
Each module represents an enrichment that can be applied to a Redis database.

## Example Usage

The following example returns a list of all modules available within your Redis Enterprise Cloud account.

```hcl-terraform
data "rediscloud_database_modules" "example" {
}

output "rediscloud_database_modules" {
  value = data.rediscloud_database_modules.example.modules
}
```

## Attributes Reference

* `modules` A list of database modules.

Each module entry provides the following attributes

* `name` The identifier assigned by the database module

* `description` A meaningful description of the database module
