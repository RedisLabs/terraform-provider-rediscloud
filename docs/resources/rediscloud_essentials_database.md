---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_essentials_database"
description: |-
  Essentials Database resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_essentials_database

This resource allows you to manage an Essentials database within your Redis Enterprise Cloud account.

-> **Note:** This is for databases within Essential Subscriptions only. See also `rediscloud_subscription_database` (Pro) and `rediscloud_active_active_subscription_database`.

## Example Usage

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_essentials_plan" "plan" {
  name           = "Multi-Zone_5GB"
  cloud_provider = "AWS"
  region         = "eu-west-1"
}

resource "rediscloud_essentials_subscription" "subscription-resource" {
  name              = "subscription-name"
  plan_id           = data.rediscloud_essentials_plan.plan.id
  payment_method_id = data.rediscloud_payment_method.card.id
}

resource "rediscloud_essentials_database" "database-resource" {
  subscription_id     = rediscloud_essentials_subscription.subscription-resource.id
  name                = "database-name"
  enable_default_user = true
  password            = "my_password"

  data_persistence = "none"
  replication      = false

  alert {
    name  = "throughput-higher-than"
    value = 80
  }
  
  tags = {
    "env" = "dev"
    "priority" = "2"
  }
}
```

## Argument Reference

The following arguments are supported:

* `subscription_id` - (Required) The ID of the subscription to create the database in. **Modifying this attribute will force creation of a new resource.**
* `name` - (Required) A meaningful name to identify the database.
* `protocol` - (Optional) Database protocol. 'stack' is a suite of all Redis' data modules. Default: 'stack'. Either: 'redis', 'memcached' or 'stack'. **'redis' is only used with Pay-As-You-Go databases.**
* `resp_version` - (Optional) RESP version must be compatible with the Redis version.
* `data_persistence` - (Required) Rate of database data persistence (in persistent storage). Either: 'none', 'aof-every-1-second', 'aof-every-write', 'snapshot-every-1-hour', 'snapshot-every-6-hours' or 'snapshot-every-12-hours'.
* `data_eviction` - (Optional) Data items eviction method. Either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru'.
* `replication` - (Required) Databases replication. Either: 'true' or 'false'.
* `periodic_backup_path` - (Optional) If specified, automatic backups will be every 24 hours and immediate backups to this path will be allowed upon request.
* `source_ips` - (Optional) List of source IP addresses or subnet masks. If specified, Redis clients will be able to connect to this database only from within the specified source IP address ranges. Example value: ['192.168.10.0/32', '192.168.12.0/24'].
* `replica` - (Optional) If specified, this database will be a replica of the specified Redis databases provided, documented below.
* `client_tls_certificates` - (Optional) A list of TLS/SSL certificates (public keys) with new line characters replaced by \n.
* `password` - (Optional) Password to access the database. If not specified, a random 32 character long alphanumeric password will be automatically generated.
* `enable_default_user` - (Optional) When `true` enables connecting to the database with the default user. Default `true`.
* `alert` - (Optional) A block defining Redis database alert. Can be specified multiple times. Documented below.
* `tags` - (Optional) A string/string map of tags to associate with this database. Note that all keys and values must be lowercase.
* `modules` - (Optional) A list of modules objects, documented below. **Modifying this attribute will force creation of a new resource.**
* `enable_payg_features` - (Optional) Whether to enable features restricted to Pay-As-You-Go legacy databases. It is not supported for new databases. Default `false`.
* `memory_limit_in_gb` - (Optional) **Only used with Pay-As-You-Go databases.** Maximum memory usage for the database.
* `support_oss_cluster_api` - (Optional) **Only used with Pay-As-You-Go databases.** Support Redis open-source (OSS) Cluster API. Default `false`.
* `external_endpoint_for_oss_cluster_api` - (Optional) **Only used with Pay-As-You-Go databases.** Should use the external endpoint for open-source (OSS) Cluster API. Default `false`.
* `enable_database_clustering` - (Optional) **Only used with Pay-As-You-Go databases.** Distributes database data to different cloud instances. Default `false`.
* `regex_rules` - (Optional) **Only used with Pay-As-You-Go databases.** Shard regex rules. Relevant only for a sharded database.
* `enable_tls` - (Optional) **Only used with Pay-As-You-Go databases.** Use TLS for authentication. Default `false`.

The `replica` block supports:

* `sync_source` - The sources to replicate. Documented below.

The `sync_source` block supports:

* `endpoint` - A Redis URI (sample format: 'redis://user:password@host:port)'. If the URI provided is a Redis Cloud instance, only the host and port should be provided (using the format: ['redis://endpoint1:6379', 'redis://endpoint2:6380'] ).
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
* `redis_version_compliance` - The compliance Redis version of the database.
* `activated_on` - When this database was activated.
* `public_endpoint` - Public endpoint to access the database.
* `private_endpoint` - Private endpoint to access the database.

## Import
`rediscloud_essentials_database` can be imported using the ID of the subscription and the ID of the database in the format {subscription ID}/{database ID}, e.g.

```
$ terraform import rediscloud_essentials_database.database-resource 123456/12345678
```
