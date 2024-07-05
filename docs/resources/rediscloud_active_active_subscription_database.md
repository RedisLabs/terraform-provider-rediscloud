---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_subscription_database"
description: |-
  Database resource for Active-Active Subscriptions in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_subscription_database

This resource allows you to manage a database within your Redis Enterprise Cloud account.

-> **Note:** This is for databases within Active-Active Subscriptions only. See also `rediscloud_subscription_database` (Pro) and `rediscloud_essentials_database`.

## Example Usage

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "subscription-resource" {
  name = "subscription-name"
  payment_method_id = data.rediscloud_payment_method.card.id 
  cloud_provider = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    region {
      region = "us-east-1"
      networking_deployment_cidr = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
    region {
      region = "us-east-2"
      networking_deployment_cidr = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 2000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "database-resource" {
    subscription_id = rediscloud_active_active_subscription.subscription-resource.id
    name = "database-name"
    memory_limit_in_gb = 1
    global_data_persistence = "aof-every-1-second"
    global_password = "some-random-pass-2" 
    global_source_ips = ["192.168.0.0/16"]
    global_alert {
      name = "dataset-size"
      value = 40
    }

    global_modules = ["RedisJSON"]

    override_region {
      name = "us-east-2"
      override_global_source_ips = ["192.10.0.0/16"]
    }

    override_region {
      name = "us-east-1"
      override_global_data_persistence = "none"
      override_global_password = "region-specific-password"
      override_global_alert {
        name = "dataset-size"
        value = 60
      }
   }
}

output "us-east-1-public-endpoints" {
  value = rediscloud_active_active_subscription_database.database-resource.public_endpoint.us-east-1
}

output "us-east-2-private-endpoints" {
  value = rediscloud_active_active_subscription_database.database-resource.private_endpoint.us-east-2
}
```

## Argument Reference

The following arguments are supported:
* `subscription_id`: (Required) The ID of the Active-Active subscription to create the database in. **Modifying this attribute will force creation of a new resource.**
* `name` - (Required) A meaningful name to identify the database. **Modifying this attribute will force creation of a new resource.**
* `memory_limit_in_gb` - (Required) Maximum memory usage for this specific database, including replication and other overhead
* `support_oss_cluster_api` - (Optional) Support Redis open-source (OSS) Cluster API. Default: ‘false’
* `external_endpoint_for_oss_cluster_api` - (Optional) Should use the external endpoint for open-source (OSS) Cluster API.
  Can only be enabled if OSS Cluster API support is enabled. Default: 'false'
* `enable_tls` - (Optional) Use TLS for authentication. Default: ‘false’
* `client_ssl_certificate` - (Optional) SSL certificate to authenticate user connections.
* `data_eviction` - (Optional) The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru')
* `global_data_persistence` - (Optional) Global rate of database data persistence (in persistent storage) of regions that dont override global settings. Default: 'none'
* `global_password` - (Optional) Password to access the database of regions that don't override global settings. If left empty, the password will be generated automatically
* `global_alert` - (Optional) A block defining Redis database alert of regions that don't override global settings, documented below, can be specified multiple times. (either: 'dataset-size', 'datasets-size', 'throughput-higher-than', 'throughput-lower-than', 'latency', 'syncsource-error', 'syncsource-lag' or 'connections-limit')
* `global_modules` - (Optional) A list of modules to be enabled on all deployments of this database. Supported modules: `RedisJSON`, `RediSearch`. Ignored after database creation.
* `global_source_ips` - (Optional) List of source IP addresses or subnet masks of regions that don't override global settings. If specified, Redis clients will be able to connect to this database only from within the specified source IP addresses ranges (example: ['192.168.10.0/32', '192.168.12.0/24'])
* `global_resp_version` - (Optional) Either 'resp2' or 'resp3'. Resp version for Crdb databases within the AA database. Must be compatible with Redis version.
* `port` - (Optional) TCP port on which the database is available - must be between 10000 and 19999. **Modifying this attribute will force creation of a new resource.**
* `override_region` - (Optional) Override region specific configuration, documented below


The `override_region` block supports:

* `name` - (Required) Region name.
* `override_global_alert` - (Optional) A block defining Redis regional instance of an Active-Active database alert, documented below, can be specified multiple times
* `override_global_password` - (Optional) If specified, this regional instance of an Active-Active database password will be used to access the database
* `override_global_source_ips` - (Optional)  List of regional instance of an Active-Active database source IP addresses or subnet masks. If specified, Redis clients will be able to connect to this database only from within the specified source IP addresses ranges (example: ['192.168.10.0/32', '192.168.12.0/24'] )
* `override_global_data_persistence` - (Optional) Regional instance of an Active-Active database data persistence rate (in persistent storage)
* `remote_backup` - (Optional) Specifies the backup options for the database in this region, documented below

The `override_global_alert` block supports:

* `name` - (Required) Alert name
* `value` - (Required) Alert value

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
* `public_endpoint` - A map of which public endpoints can to access the database per region, uses region name as key.
* `private_endpoint` - A map of which private endpoints can to access the database per region, uses region name as key.
* `override_region.latest_backup_status` - On each override_region block, the latest_backup_status is reported, an object documented below.
* `latest_import_status` - A latest_import_status object, documented below.

The `latest_backup_status` object and `latest_import_status` block contains:

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

## Import
`rediscloud_active_active_subscription_database` can be imported using the ID of the Active-Active subscription and the ID of the database in the format {subscription ID}/{database ID}, e.g.

```
$ terraform import rediscloud_active_active_subscription_database.database-resource 123456/12345678
```

Note: Due to constraints in the Redis Cloud API, the import process will not import global attributes or override region attributes. If you wish to use these attributes in your Terraform configuration, you will need to manually add them to your Terraform configuration and run `terraform apply` to update the database.

