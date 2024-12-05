---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription_database"
description: |-
  Database resource in the Redis Cloud Terraform provider
---

# Resource: rediscloud_subscription_database

This resource allows you to manage a database within your Redis Enterprise Cloud account.

-> **Note:** This is for databases within Pro Subscriptions only. See also `rediscloud_active_active_subscription_database` and `rediscloud_essentials_database`.

## Example Usage

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "subscription-resource" {

  name = "subscription-name"
  payment_method = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    region {
      region = "eu-west-1"
      multiple_availability_zones = true
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["euw1-az1", "euw1-az2", "euw1-az3"]
    }
  }

  // This block needs to be defined for provisioning a new subscription.
  // This allows creation of a well-optimized hardware specification for databases in the cluster
  creation_plan {
    dataset_size_in_gb = 15
    quantity = 1
    replication=true
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 20000
    modules = ["RedisJSON"]
  }
}

// The primary database to provision
resource "rediscloud_subscription_database" "database-resource" {
    subscription_id = rediscloud_subscription.subscription-resource.id
    name = "database-name"
    dataset_size_in_gb = 15
    data_persistence = "aof-every-write"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 20000
    replication = true
  
    modules = [
        {
          name = "RedisJSON"
        }
    ]
    
    alert {
      name = "dataset-size"
      value = 40
    }
  
    tags = {
      "market" = "emea"
    }
}
```

## Argument Reference

The following arguments are supported:

* `subscription_id` - (Required) The ID of the subscription to create the database in. **Modifying this attribute will force creation of a new resource.**
* `name` - (Required) A meaningful name to identify the database
* `throughput_measurement_by` - (Required) Throughput measurement method that will be used by your databases. Either `number-of-shards` or `operations-per-second`. **`number-of-shards` is deprecated and only supported for legacy deployments.**
* `throughput_measurement_value` - (Required) Throughput value (as applies to selected measurement method)
* `memory_limit_in_gb` - (Optional -  **Required if `dataset_size_in_gb` is unset**) Maximum memory usage for this specific database, including replication and other overhead **Deprecated in favor of `dataset_size_in_gb` - not possible to import databases with this attribute set**
* `dataset_size_in_gb` - (Optional - **Required if `memory_limit_in_gb` is unset**) The maximum amount of data in the dataset for this specific database is in GB
* `protocol` - (Optional) The protocol that will be used to access the database, (either ‘redis’ or ‘memcached’) Default: ‘redis’. **Modifying this attribute will force creation of a new resource.**
* `support_oss_cluster_api` - (Optional) Support Redis open-source (OSS) Cluster API. Default: ‘false’
* `resp_version` - (Optional) Either `resp2` or `resp3`. Database's RESP version. Must be compatible with the Redis version.
* `external_endpoint_for_oss_cluster_api` - (Optional) Should use the external endpoint for open-source (OSS) Cluster API.
  Can only be enabled if OSS Cluster API support is enabled. Default: 'false'
* `client_ssl_certificate` - (Optional) SSL certificate to authenticate user connections, conflicts with `client_tls_certificates`
* `client_tls_certificates` - (Optional) A list of TLS certificates to authenticate user connections, conflicts with `client_ssl_certificate`
* `periodic_backup_path` - (Optional) Path that will be used to store database backup files. **Deprecated: Use `remote_backup` block instead**
* `replica_of` - (Optional) Set of Redis database URIs, in the format `redis://user:password@host:port`, that this
  database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided.
  Cannot be enabled when `support_oss_cluster_api` is enabled.
* `modules` - (Optional) A list of modules objects, documented below. **Modifying this attribute will force creation of a new resource.**
* `alert` - (Optional) A block defining Redis database alert, documented below, can be specified multiple times
* `data_persistence` - (Optional) Rate of database's storage data persistence (either: 'none', 'aof-every-1-second', 'aof-every-write', 'snapshot-every-1-hour', 'snapshot-every-6-hours' or 'snapshot-every-12-hours'). Default: ‘none’
* `data_eviction` - (Optional) The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'). Default: 'volatile-lru'
* `password` - (Optional) Password to access the database. If omitted, a random 32 character long alphanumeric password will be automatically generated
* `replication` - (Optional) Databases replication. Default: ‘true’
* `average_item_size_in_bytes` - (Optional) Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes)
  of the items stored in the database. Default: 1000
* `source_ips` - (Optional) List of source IP addresses or subnet masks. If specified, Redis clients will be able to connect to this database only from within the specified source IP addresses ranges (example: [‘192.168.10.0/32’, ‘192.168.12.0/24’])
* `hashing_policy` - (Optional) List of regular expression rules to shard the database by. See
  [the documentation on clustering](https://docs.redislabs.com/latest/rc/concepts/clustering/) for more information on the
  hashing policy. This cannot be set when `support_oss_cluster_api` is set to true.
* `enable_tls` - (Optional) Use TLS for authentication. Default: ‘false’
* `port` - (Optional) TCP port on which the database is available - must be between 10000 and 19999. **Modifying this attribute will force creation of a new resource.**
* `remote_backup` (Optional) Specifies the backup options for the database, documented below
* `enable_default_user` (Optional) When `true` enables connecting to the database with the default user. Default `true`.
* `tags` - (Optional) A string/string map of Tags to associate with this database. Note that all keys and values must be lowercase.

The `alert` block supports:

* `name` (Required) - Alert name. (either: 'dataset-size', 'datasets-size', 'throughput-higher-than', 'throughput-lower-than', 'latency', 'syncsource-error', 'syncsource-lag' or 'connections-limit') 
* `value` (Required) - Alert value

The `modules` list supports:

* `name` (Required) - Name of the Redis database module to enable. **Modifying this attribute will force creation of a new resource.**

  Example:
  
  ```hcl
    modules = [
        {
          "name": "RedisJSON"
        },
        {
          "name": "RedisBloom"
        }
    ]
  ```

The `remote_backup` block supports:

* `interval` (Required) - Defines the interval between backups. Should be in the following format 'every-x-hours'. x is one of [24,12,6,4,2,1]. For example: 'every-4-hours'
* `time_utc` (Optional) - Defines the hour automatic backups are made - only applicable when the interval is `every-12-hours` or `every-24-hours`. For example: '14:00'
* `storage_type` (Required) - Defines the provider of the storage location
* `storage_path` (Required) - Defines a URI representing the backup storage location

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 30 mins) Used when creating the database
* `update` - (Defaults to 30 mins) Used when updating the database
* `delete` - (Defaults to 10 mins) Used when destroying the database

## Attribute reference

* `db_id` - Identifier of the database created
* `public_endpoint` - Public endpoint to access the database
* `private_endpoint` - Private endpoint to access the database

## Import
`rediscloud_subscription_database` can be imported using the ID of the subscription and the ID of the database in the format {subscription ID}/{database ID}, e.g.

```
$ terraform import rediscloud_subscription_database.database-resource 123456/12345678
```

Note: Due to constraints in the Redis Cloud API, the `memory_limit_in_gb` cannot be set during imports as it is deprecated. If you need to set the `memory_limit_in_gb` attribute, you will need to create a new database resource. It is recommended to use the `dataset_size_in_gb` attribute instead.
