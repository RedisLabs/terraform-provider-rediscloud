---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_essentials_database"
description: |-
  Essentials Database resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_essentials_database

Creates an Essentials Database within a specified Essentials Subscription in your Redis Enterprise Cloud Account.

## Example Usage

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_essentials_plan" "example" {
  name = "250MB"
  cloud_provider = "AWS"
  region = "eu-west-1"
}

resource "rediscloud_essentials_subscription" "example" {
  name = "%s"
  plan_id = data.rediscloud_essentials_plan.example.id
  payment_method_id = data.rediscloud_payment_method.card.id
}

resource "rediscloud_essentials_database" "example" {
  subscription_id = rediscloud_essentials_subscription.example.id
  name = "%s"
  enable_default_user = true
  password = "my_password"

  alert {
    name = "throughput-higher-than"
    value = 80
  }
}
```

## Argument Reference

The following arguments are supported:

* `subscription_id` - (Required) The ID of the subscription to create the database in. **Modifying this attribute will force creation of a new resource.**
* `name` - (Required) A meaningful name to identify the database.
* `protocol` - (Optional) The protocol that will be used to access the database, (either ‘redis’, 'memcached’ or 'stack'). **Modifying this attribute will force creation of a new resource.**
* `resp_version` - (Optional) RESP version must be compatible with Redis version.
* `data_persistence` - (Optional) Rate of database data persistence (in persistent storage). Default: 'none'.
* `data_eviction` - (Optional) Data items eviction method.
* `replication` - (Optional) Databases replication. Default: 'false'.
* `periodic_backup_path` - (Optional) If specified, automatic backups will be every 24 hours or database will be able to perform immediate backups to this path.
* `source_ips` - (Optional) List of source IP addresses or subnet masks. If specified, Redis clients will be able to connect to this database only from within the specified source IP addresses ranges. example value: ['192.168.10.0/32', '192.168.12.0/24'].
* `replica` - (Optional) If specified, this database will be a replica of the specified Redis databases provided, documented below.
* `client_tls_certificates` - (Optional) A list of TLS/SSL certificates (public keys) with new line characters replaced by \n.
* `password` - (Optional) Password to access the database. If omitted, a random 32 character long alphanumeric password will be automatically generated.
* `enable_default_user` - (Optional) When `true` enables connecting to the database with the default user. Default `true`.
* `alert` - (Optional) A block defining Redis database alert, documented below, can be specified multiple times.
* `modules` - (Optional) A list of modules objects, documented below. **Modifying this attribute will force creation of a new resource.**
* `enable_payg_features` - (Optional) Whether to enable features restricted to Pay-As-You-Go databases. Newly created databases should omit this property. Default `false`.
* `memory_limit_in_gb` - (Optional) **Only used with Pay-As-You-Go databases.** Maximum memory usage for this specific database.
* `support_oss_cluster_api` - (Optional) **Only used with Pay-As-You-Go databases.** Support Redis open-source (OSS) Cluster API. Default `false`.
* `external_endpoint_for_oss_cluster_api` - (Optional) **Only used with Pay-As-You-Go databases.** Should use the external endpoint for open-source (OSS) Cluster API. Default `false`.
* `enable_database_clustering` - (Optional) **Only used with Pay-As-You-Go databases.** Distributes database data to different cloud instances. Default `false`.
* `regex_rules` - (Optional) **Only used with Pay-As-You-Go databases.** Shard regex rules. Relevant only for a sharded database.
* `enable_tls` - (Optional) **Only used with Pay-As-You-Go databases.** Use TLS for authentication. Default `false`.

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

## Attribute Reference

* `cloud_provider` - The Cloud Provider hosting this database.
* `region` - The region within the Cloud Provider where this database is hosted.
* `redis_version_compliance` - The compliance level (redis version) of this database.
* `activated_on` - When this database was activated.
* `public_endpoint` - Public endpoint to access the database.
* `private_endpoint` - Private endpoint to access the database.
* `latest_backup_status` - A latest_backup_status object, documented below.
* `latest_import_status` - A latest_import_status object, documented below.

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
