---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_subscription_regions"
description: |-
  Active-Active subscription regions data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_active_active_subscription_regions

The Active-Active subscription regions data source allows access to the regions associated with an Active-Active subscription within your Enterprise Cloud Account.

## Example Usage

```terraform

data "rediscloud_active_active_subscription_regions" "example" {
  subscription_name = rediscloud_active_active_subscription.example.name
}

output "rediscloud_active_active_subscription_regions" {
  value = data.rediscloud_active_active_subscription_regions.example.regions
}
```

## Argument Reference

* `subscription_name` - (Required) The name of the g subscription.

## Attribute Reference

* `subscription_name` - The name of the subscription.
* `regions` - A list of regions associated with an Active-Active subscription.

Each block within the `regions` list supports:

* `region_id` - The unique identifier of the region.
* `region` - Deployment region as defined by the cloud provider.
* `networking_deployment_cidr` - Deployment CIDR mask.
* `vpc_id` - VPC ID for the region.
* `databases` - A list of databases found in the region.

The `databases` block supports:

* `database_id` - A numeric ID for the database.
* `database_name` - The name of the database.
* `write_operations_per_second` - Write operations per second for the database.
* `read_operations_per_second` - Read operations per second for the database.
