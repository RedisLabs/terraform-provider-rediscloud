---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_subscription_regions"
description: |-
  Regions resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_subscription_regions

Manages regions within your Redis Enterprise Cloud Active-Active subscription.
This resource is responsible for creating and managing regions within that subscription.
This allows Redis Enterprise Cloud to efficiently provision your cluster within each defined region in a separate block.

## Example Usage

```hcl  
resource "rediscloud_active_active_subscription_regions" "regions-resource" {
	subscription_id = rediscloud_active_active_subscription.subscription-resource.id
	delete_regions = false
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "192.168.0.0/24" 
	  database {
		  database_id = rediscloud_active_active_subscription_database.database-resource.db_id
      database_name = rediscloud_active_active_subscription_database.database-resource.name
		  local_write_operations_per_second = 1000
		  local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "us-east-2"
	  networking_deployment_cidr = "10.0.1.0/24"
    local_resp_version = "resp2"
	  database {
		  database_id = rediscloud_active_active_subscription_database.database-resource.db_id
      database_name = rediscloud_active_active_subscription_database.database-resource.name
		  local_write_operations_per_second = 2000
		  local_read_operations_per_second = 4000
	  }
	}
 }
```

## Argument Reference

The following arguments are supported:

* `subscription_id` - (Required) ID of the subscription that the regions belong to. **Modifying this attribute will force creation of a new resource.**
* `delete_regions` - (Optional) Flag required to be set when one or more regions is to be deleted, if the flag is not set an error will be thrown
* `region` - (Required) Cloud networking details, per region, documented below

The `region` block supports:

* `region_id` - (Computed) The ID of the region, as created by the API
* `region` - (Required) Region name
* `vpc_id` - (Computed) Identifier of the VPC to be peered, set by the API
* `networking_deployment_cidr` - (Required) Deployment CIDR mask. The total number of bits must be 24 (x.x.x.x/24)
* `recreate_region` - (Optional) Protection flag, needs to be set if a region has to be re-created. A region will need to be re-created in the case of a change on the `networking_deployment_cidr` field. During re-create, the region will be deleted (so the `delete_regions` flag also needs to be set) and then created again. Default: 'false'
* `local_resp_version` - (Optional) Either 'resp2' or 'resp3'. Resp version for Crdb databases within this region. Must be compatible with Redis version.
* `database` - (Required) A block defining the write and read operations in the region, per database, documented below

The `database` block supports:

* `database_id` - (Required) Database ID belonging to the subscription
* `database_name` - (Required) Database name belonging to the subscription
* `local_write_operations_per_second` - (Required) Local write operations per second for this active-active region
* `local_read_operations_per_second` - (Required) Local read operations per second for this active-active region

## Attribute Reference

* `latest_backup_status` - An object containing the JSON-formatted response detailing the latest backup status (or an error if the lookup failed).

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 60 mins) Used when creating the regions
* `update` - (Defaults to 60 mins) Used when updating the regions
* `delete` - (Defaults to 10 mins) Used when destroying the regions

## Import

`rediscloud_active_active_regions` can be imported using the ID of the subscription, e.g.

```
$ terraform import rediscloud_active_active_regions.regions-resource 12345678
```

