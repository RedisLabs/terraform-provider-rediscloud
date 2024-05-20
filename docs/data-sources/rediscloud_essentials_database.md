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

* `protocol` - The protocol of the database. Either `redis`, `memcached` or `stack`
* `cloud_provider` - The Cloud Provider hosting this database
* `region` - The region within the Cloud Provider where this database is hosted
* `redis_version_compliance` - The compliance level (redis version) of this database.
* `resp_version` - Either `resp2` or `resp3`. Database's RESP version
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
* `latest_backup_status` - An object containing the JSON-formatted response detailing the latest backup status (or an error if the lookup failed).
* `latest_import_status` - An object containing the JSON-formatted response detailing the latest import status (or an error if the lookup failed).

The `replica` block supports:

* `sync_source` - The sources to replicate, documented below.

The `sync_source` block supports:]

* `endpoint` - A Redis URI (sample format: 'redis://user:password@host:port)'. If the URI provided is Redis Cloud instance, only host and port should be provided (using the format: ['redis://endpoint1:6379', 'redis://endpoint2:6380'] ).
* `encryption` - Defines if encryption should be used to connect to the sync source.
* `server_cert` - TLS certificate chain of the sync source.

The `alert` block supports:

* `name` - The alert name
* `value` - The alert value

Each `modules` entry provides the following attributes

* `name` - The identifier assigned by the database module.
