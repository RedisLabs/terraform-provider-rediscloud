---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription_database"
description: |-
  Database resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_subscription_database

Creates a Database within a specified Subscription in your Redis Enterprise Cloud Account.

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
      preferred_availability_zones = ["euw1-az1, euw1-az2, euw1-az3"]
    }
  }

  // This block needs to be defined for provisioning a new subscription.
  // This allows creating a well-optimised hardware specification for databases in the cluster
  creation_plan {
    average_item_size_in_bytes = 1
    memory_limit_in_gb = 1
    quantity = 2
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = ["RedisJSON", "RedisBloom"]
  }
}

// The primary database to provision
resource "rediscloud_subscription_database" "database-resource" {
    subscription_id = rediscloud_subscription.subscription-resource.id
    name = "database-name"
    protocol = "redis"
    memory_limit_in_gb = 1
    data_persistence = "none"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    support_oss_cluster_api = false 
    external_endpoint_for_oss_cluster_api = false
    replication = true
   
    modules = [
        {
          name = "RedisJSON"
        },
        {
          name = "RedisBloom"
        }
    ]
    
    alert {
      name = "dataset-size"
      value = 40
    }
    depends_on = [rediscloud_subscription.subscription-resource]

}

// An example of how a replica database can be provisioned
resource "rediscloud_subscription_database" "database-resource-replica" {
    subscription_id = rediscloud_subscription.subscription-resource.id
    name = "database-name-replica"
    protocol = "redis"
    memory_limit_in_gb = 1
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    replica_of = [format("redis://%s", rediscloud_subscription_database.database-resource.public_endpoint)]
    depends_on = [rediscloud_subscription.subscription-resource]

} 
```

## Argument Reference

The following arguments are supported:

* `subscription_id` - (Required) The ID of the subscription to create the database in. **Modifying this attribute will force creation of a new resource.**
* `name` - (Required) A meaningful name to identify the database
* `throughput_measurement_by` - (Required) Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)
* `throughput_measurement_value` - (Required) Throughput value (as applies to selected measurement method)
* `memory_limit_in_gb` - (Required) Maximum memory usage for this specific database
* `protocol` - (Optional) The protocol that will be used to access the database, (either ‘redis’ or ‘memcached’) Default: ‘redis’. **Modifying this attribute will force creation of a new resource.**
* `support_oss_cluster_api` - (Optional) Support Redis open-source (OSS) Cluster API. Default: ‘false’
* `external_endpoint_for_oss_cluster_api` - (Optional) Should use the external endpoint for open-source (OSS) Cluster API.
  Can only be enabled if OSS Cluster API support is enabled. Default: 'false'
* `client_ssl_certificate` - (Optional) SSL certificate to authenticate user connections
* `periodic_backup_path` - (Optional) Path that will be used to store database backup files
* `replica_of` - (Optional) Set of Redis database URIs, in the format `redis://user:password@host:port`, that this
  database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided.
  Cannot be enabled when `support_oss_cluster_api` is enabled.
* `modules` - (Optional) A list of modules objects, documented below. **Modifying this attribute will force creation of a new resource.**
* `alert` - (Optional) A block defining Redis database alert, documented below, can be specified multiple times
* `data_persistence` - (Optional) Rate of database data persistence (in persistent storage). Default: ‘none’
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

The `alert` block supports:

* `name` (Required) - Alert name
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

* `interval` (Required) - Defines the frequency of the automatic backup
* `time_utc` (Optional) - Defines the hour automatic backups are made - only applicable when interval is `every-12-hours` or `every-24-hours`
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

