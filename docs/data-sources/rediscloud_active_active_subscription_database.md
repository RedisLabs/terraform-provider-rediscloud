---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_subscription_database"
description: |-
  Database data source for Active-Active Subscriptions in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_subscription_database

This data source allows access to the details of an existing database within your Redis Enterprise Cloud account.

-> **Note:** This is for databases within Active-Active Subscriptions only. See also `rediscloud_database` (Pro) and `rediscloud_essentials_database`.

## Example Usage

The following example shows how to locate a single database within an AA Subscription. This example assumes the subscription
only contains a single database.

```hcl-terraform
data "rediscloud_active_active_subscription_database" "example" {
  subscription_id = "1234"
}
```

The following example shows how to use the name to locate a single database within an AA subscription that has multiple
databases.

```hcl-terraform
data "rediscloud_active_active_subscription_database" "example" {
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
* `dataset_size_in_gb` - Maximum amount of data in the dataset for this specific database in GB.
* `support_oss_cluster_api` - Supports the Redis open-source (OSS) Cluster API.
* `external_endpoint_for_oss_cluster_api` - Use the external endpoint for open-source (OSS) Cluster API.
* `enable_tls` - Enable TLS for database.
* `data_eviction` - The data items eviction policy.
* `global_modules` - A list of modules to be enabled on all deployments of this database.
* `public_endpoint` - Public endpoint to access the database.
* `private_endpoint` - Private endpoint to access the database.
* `latest_backup_statuses` A list of `latest_backup_status` objects, documented below.
* `latest_import_status` - A `latest_import_status` object, documented below.`
* `tags` - A string/string map of all Tags associated with this database.

The `latest_backup_status` block contains:

* `region` - The region within the Cloud Provider where this database is hosted.
* `error` - An error block, in case this lookup failed, documented below.
* `response` - A detail block, documented below.

The `latest_import_status` block contains:

* `error` - An error block, in case this lookup failed, documented below.
* `response` - A detail block, documented below.

The `error` block in `latest_backup_status` and `latest_import_status` contains:

* `type` - The type of error encountered while looking up the status of the last import.
* `description` - A description of the error encountered while looking up the status of the last import.
* `status` - Any particular HTTP status code associated with the erroneous status check.

The `response` block `latest_backup_status` contains:

* `status` - The status of the last backup operation.
* `last_backup_time` - When the last backup operation occurred.
* `failure_reason` - If a failure, why the backup operation failed.

The `response` block `latest_import_status` contains:

* `status` - The status of the last import operation.
* `last_import_time` - When the last import operation occurred.
* `failure_reason` - If a failure, why the import operation failed.
* `failure_reason_params` - Parameters of the failure, if appropriate, in the form of a list of objects each with a `key` entry and a `value` entry.
