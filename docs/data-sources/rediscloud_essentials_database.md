---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_essentials_database"
description: |-
  Database data source for Essentials Subscriptions in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_essentials_database

The Essentials Database data source allows access to the details of an existing database within your Redis Enterprise
Cloud account.

## Example Usage

The following example shows how to locate a single database within a Subscription.  This example assumes the subscription only contains a single database.

```hcl-terraform
data "rediscloud_essentials_database" "example" {
  subscription_id = "1234"
}
```

The following example shows how to use the name to locate a single database within a subscription that has multiple databases.

```hcl-terraform
data "rediscloud_essentials_database" "example" {
  subscription_id = "1234"
  name = "first-database"
}
```

## Argument Reference

* `subscription_id` - (Required) ID of the subscription that the database belongs to
* `db_id` - (Optional) The id of the database to filter returned databases
* `name` - (Optional) The name of the database to filter returned databases

## Attribute Reference

* `protocol` - The protocol of the database. Either `redis`, `memcached` or `stack`.
* `cloud_provider` - The Cloud Provider hosting this database.
* `region` - The region within the Cloud Provider where this database is hosted.
* `redis_version_compliance` - The compliance level (redis version) of this database.
* `resp_version` - Either `resp2` or `resp3`. Database's RESP version.
* `data_persistence` - The rate of database data persistence (in persistent storage).
* `data_eviction` - The data items eviction policy.
* `replication` - Database replication.
* `activated_on` - When this database was activated.
* `periodic_backup_path` - Automatic backups will be every 24 hours or database will be able to perform immediate backups to this path.
* `public_endpoint` - Public endpoint to access the database.
* `private_endpoint` - Private endpoint to access the database.
* `source_ips` - Set of CIDR addresses to allow access to the database.
* `replica` - Replica details on this database, documented below.
* `client_tls_certificates` - A list of TLS certificates (public keys) with new line characters replaced by \n.
* `password` - The password used to access the database - not present on `memcached` protocol databases.
* `enable_default_user` - When `true` enables connecting to the database with the default user.
* `alert` - Set of alerts to enable on the database, documented below.
* `modules` A list of database modules, documented below.
* `latest_backup_status` - A latest_backup_status object, documented below.
* `latest_import_status` - A latest_import_status object, documented below.

The `replica` block supports:

* `sync_source` - The sources to replicate, documented below.

The `sync_source` block supports:

* `endpoint` - A Redis URI (sample format: 'redis://user:password@host:port)'. If the URI provided is Redis Cloud instance, only host and port should be provided (using the format: ['redis://endpoint1:6379', 'redis://endpoint2:6380'] ).
* `encryption` - Defines if encryption should be used to connect to the sync source.
* `server_cert` - TLS certificate chain of the sync source.

The `alert` block supports:

* `name` - The alert name.
* `value` - The alert value.

Each `modules` entry provides the following attributes:

* `name` - The identifier assigned by the database module.

The `latest_backup_status` and `latest_import_status` blocks contain:

* `error` - An error block, in case this lookup failed, documented below.
* `response` - A detail block, documented below.

The `error` block in both `latest_backup_status` and `latest_import_status` contains:

* `type` - The type of error encountered while looking up the status of the last backup/import.
* `description` - A description of the error encountered while looking up the status of the last backup/import.
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
