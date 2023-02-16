---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_regions"
description: |-
  Regions resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_active_active_regions

Creates an Active Active Regions within your Redis Enterprise Cloud Account.
This resource is responsible for creating regions within
that subscription. This allows Redis Enterprise Cloud to provision
your regions defined in separate resources in the most efficient way.

## Example Usage

```hcl
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}
  
resource "rediscloud_active_active_subscription_regions" "example" {
	subscription_id = 151945
	delete_regions = false
	region {
	  region = "us-east-1"
	  networking_deployment_cidr = "10.0.0.0/24" 
	  recreate_region = false
	  database {
		id = "7839"
		database_name = "test-db-1"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
	  region = "eu-west-1"
	  networking_deployment_cidr = "10.1.0.0/24" 
	  recreate_region = false
	  database {
		id = "7839"
		database_name = "test-db-1"
		local_write_operations_per_second = 1000
		local_read_operations_per_second = 1000
	  }
	}
	region {
		region = "eu-west-2"
		networking_deployment_cidr = "10.2.0.0/24" 
		recreate_region = false
		database {
		  id = "7839"
		  database_name = "test-db-1"
		  local_write_operations_per_second = 1500
		  local_read_operations_per_second = 1500
		}
	  }
 }
```

## Argument Reference

The following arguments are supported:

* `subscription_id` - (Required) ID of the subscription that the regions belong to
* `delete_regions` - (Optional) Flag required to be set when one or more regions is to be deleted, if the flag is not set an error will be thrown
* `region` - (Required) Cloud networking details, per region, documented below

The `region` block supports:

* `region` - (Required) Region name
* `vpc_id` - (Required) Identifier of the VPC to be peered
* `networking_deployment_cidr` - (Required) Deployment CIDR mask
* `recreate_region` - (Optional) Flag, needs to be set if a region has to be re-created. A region will need to be re-created in the case of a change on 
  the `networking_deployment_cidr` field. During re-create the region will be deleted (so the `delete_regions` flag also needs to be set) and then created again.
* `database` - (Required) The database resource, documented below

The `database` block supports:

* `id` - (Required) Database id beloging to the subscription
* `database_name` - (Required) Database name belonging to the subscription
* `local_write_operations_per_second` - (Required) Local throughput measurement for an active-active region
* `local_read_operations_per_second` - (Required) Local throughput measurement for an active-active region


### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 60 mins) Used when creating the subscription
* `update` - (Defaults to 60 mins) Used when updating the subscription
* `delete` - (Defaults to 10 mins) Used when destroying the subscription

## Import

`rediscloud_active_active_regions` can be imported using the ID of the subscription, e.g.

```
$ terraform import rediscloud_active_active_regions.example 12345678
```

~> **Note:** the creation_plan block will be ignored during imports.
