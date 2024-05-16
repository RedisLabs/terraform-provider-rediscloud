---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_database"
description: |-
  Database data source for Active-Active Subscriptions in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_database

The Active Active Database data source allows access to the details of an existing database within your Redis Enterprise
Cloud account.

## Example Usage

The following example shows how to locate a single database within an AA Subscription. This example assumes the subscription
only contains a single database.

```hcl-terraform
data "rediscloud_active_active_database" "example" {
  subscription_id = "1234"
}
```

The following example shows how to use the name to locate a single database within an AA subscription that has multiple
databases.

```hcl-terraform
data "rediscloud_active_active_database" "example" {
  subscription_id = "1234"
  name            = "first-database"
}
```

## Argument Reference

* `subscription_id` - (Required) The AA subscription to which the database belongs
* `db_id` - (Optional) The id of the database to filter returned subscriptions
* `name` - (Optional) The name of the database to filter returned subscriptions

## Attribute reference

`id` is set to the ID of the found subscription and database in the following format: `{subscription_id}/{db_id}`

* `memory_limit_in_gb` - The maximum memory usage for the database.
* `support_oss_cluster_api` - Supports the Redis open-source (OSS) Cluster API.
* `external_endpoint_for_oss_cluster_api` - Use the external endpoint for open-source (OSS) Cluster API.
* `enable_tls` - Enable TLS for database.
* `data_eviction` - The data items eviction policy.
* `global_modules` - A list of modules to be enabled on all deployments of this database.
* `public_endpoint` - Public endpoint to access the database.
* `private_endpoint` - Private endpoint to access the database.
* `latest_import_status` - An object containing the JSON-formatted response detailing the latest import status (or an error if the lookup failed).
