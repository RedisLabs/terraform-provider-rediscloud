---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_database"
description: |-
  Database data source in the Terraform provider Redis Cloud.
---

# Data Source: rediscloud_database

The Database data source allows access to the details of an existing database within your Redis Enterprise Cloud account.

## Example Usage

The following example shows how to locate a single database within a Subscription.  This example assumes the subscription only contains a single database.

```hcl-terraform
data "rediscloud_database" "example" {
  subscription_id = "1234"
}
```

The following example shows how to use the name to locate a single database within a subscription that has multiple databases. 

```hcl-terraform
data "rediscloud_database" "example" {
  subscription_id = "1234"
  name = "first-database"
}
```


## Argument Reference

* `subscription_id` - (Required) ID of the subscription that the database belongs to
* `name` - (Optional) The name of the database to filter returned databases
* `protocol` - (Optional) The protocol of the database to filter returned databases
* `region` - (Optional) The region of the database to filter returned databases

## Attributes Reference

* `name` - The name of the database
* `protocol` - The protocol of the database.
* `memory_limit_in_gb` - The maximum memory usage for the database.
* `support_oss_cluster_api` - Supports the Redis open-source (OSS) Cluster API.
* `replica_of` - The set of Redis database URIs, in the format `redis://user:password@host:port`, that this
database will be a replica of.
* `alert` - Set of alerts to enable on the database, documented below.
* `data_persistence` - The rate of database data persistence (in persistent storage).
* `password` - The password used to access the database - not present on `memcached` protocol databases.
* `replication` - Database replication.
* `throughput_measurement_by` - The throughput measurement method.
* `throughput_measurement_value` - The throughput value.
* `hashing_policy` - The list of regular expression rules the database is sharded by. See
[the documentation on clustering](https://docs.redislabs.com/latest/rc/concepts/clustering/) for more information on the
hashing policy.
* `public_endpoint` - Public endpoint to access the database
* `private_endpoint` - Private endpoint to access the database

The `alert` block supports:

* `name` The alert name
* `value` The alert value
