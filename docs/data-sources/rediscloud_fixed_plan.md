---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_fixed_plan"
description: |-
  Fixed Plan data source in the Terraform provider Redis Cloud.
---

# Data Source: rediscloud_fixed_plan

The Fixed Plan data source allows access to the templates for Fixed Subscriptions. 

## Example Usage

```hcl
data "rediscloud_fixed_plan" "example" {
  name = "Single-Zone_1GB"
  provider = "AWS"
  region = "us-west-1"
}

output "rediscloud_fixed_plan" {
  value = data.rediscloud_fixed_plan.example.id
}
```

## Argument Reference

* `id` - (Optional) The Plan's unique identifier. If you know it, this is all you need.
* `name` - (Optional) A convenient name for the plan. Not guaranteed to be unique, especially across provider/region.
* `size` - (Optional) The capacity of databases created in this plan.
* `size_measurement_unit` - (Optional) The units of 'size', usually 'MB' or 'GB'.
* `cloud_provider` - (Optional) The cloud provider: 'AWS', 'GCP' or 'Azure' (case sensitive).
* `region` - (Optional) The region to place databases in, format and availability dependent on cloud_provider.
* `region_id` - (Optional) An internal, unique-across-cloud-providers id for region.
* `availability` - (Optional) 'No replication', 'Single-zone' or 'Multi-zone'.
* `support_data_persistence` - (Optional) Whether or not databases created under Subscriptions from this Plan will support Data Persistence.
* `support_instant_and_daily_backups` - (Optional) ... will support backups.
* `support_replication` - (Optional) ... will support replication.
* `support_clustering` - (Optional) ... will support clustering

## Attribute reference

* `id` - The Plan's unique identifier. You likely only want this value.
* `name` - A convenient name for the plan.
* `size` - The capacity of databases created in this plan.
* `size_measurement_unit` - The units of 'size', usually 'MB' or 'GB'.
* `cloud_provider` - The cloud provider: 'AWS', 'GCP' or 'Azure'.
* `region` - The region databases are placed in, format and availability dependent on cloud_provider.
* `region_id` - An internal, unique-across-cloud-providers id for region.
* `price` - The plan's cost.
* `price_currency` - Self-explanatory.
* `price_period` - Self-explanatory, usually 'Month'.
* `maximum_databases` - Self-explanatory.
* `maximum_throughput` - Self-explanatory.
* `maximum_bandwidth_in_gb` - Self-explanatory.
* `availability` - 'No replication', 'Single-zone' or 'Multi-zone'.
* `connections` - Self-explanatory.
* `cidr_allow_rules` - Self-explanatory.
* `support_data_persistence` - Whether or not databases created under Subscriptions from this Plan will support Data Persistence.
* `support_instant_and_daily_backups` - ... will support backups.
* `support_replication` - ... will support replication.
* `support_clustering` - ... will support clustering.
* `supported_alerts` - a list of the alerts supported by databases created under Subscriptions from this Plan.
* `customer_support` - Level of customer support available e.g. 'Basic', 'Standard'.
