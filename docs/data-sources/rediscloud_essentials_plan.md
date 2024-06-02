---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_essentials_plan"
description: |-
  Essentials Plan data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_essentials_plan

The Essentials Plan data source allows access to the templates for Essentials Subscriptions. 
To retrieve all available plans, use [Redis Cloud API](https://api.redislabs.com/v1/swagger-ui/index.html#/Subscriptions%20-%20Fixed/getAllFixedSubscriptionsPlans).

## Example Usage

```hcl
data "rediscloud_essentials_plan" "plan" {
  name = "Single-Zone_1GB"
  provider = "AWS"
  region = "us-west-1"
}

output "rediscloud_essentials_plan" {
  value = data.rediscloud_essentials_plan.plan.id
}
```

## Argument Reference

* `id` - (Optional) The Plan's unique identifier. If known, there is no need to specify other parameters.
* `name` - (Optional) The plan's name. Not guaranteed to be unique across providers/regions.
* `size` - (Optional) The capacity of databases created in this plan.
* `size_measurement_unit` - (Optional) The units of 'size'. Either 'MB' or 'GB'.
* `cloud_provider` - (Optional) The cloud provider. Either: 'AWS', 'GCP' or 'Azure'.
* `region` - (Optional) The region in which to place the database. The format and availability are dependent on 'cloud_provider'.
* `availability` - (Optional) Either 'No replication', 'Single-zone' or 'Multi-zone'.
* `support_data_persistence` - (Optional) The plan persistence support. If 'true', you define the persistence rate on the database level.
* `support_replication` - (Optional) Databases replication support. Either 'true' or 'false'.

## Attribute reference

* `id` - The Plan's unique identifier. If known, there is no need to specify other parameters.
* `name` - The plan's name. Not guaranteed to be unique across providers/regions.
* `size` - The capacity of databases created in this plan.
* `size_measurement_unit` - The units of 'size'. Either 'MB' or 'GB'.
* `cloud_provider` - The cloud provider. Either: 'AWS', 'GCP' or 'Azure'.
* `region` - The region in which to place the database. The format and availability are dependent on 'cloud_provider'.
* `region_id` - An internal, unique region id.
* `price` - The plan's cost.
* `price_currency` - Price currency.
* `price_period` - Price period. Usually 'month'.
* `maximum_databases` - The maximum amount of databases the plan supports.
* `maximum_throughput` - The maximum throughput the plan supports.
* `maximum_bandwidth_in_gb` - The maximum network bandwidth the plan supports.
* `availability` - Either 'No replication', 'Single-zone' or 'Multi-zone'.
* `connections` - The maximum allowed connections of the plan.
* `cidr_allow_rules` - Self-explanatory.
* `support_data_persistence`— The plan persistence support. If 'true', you define the persistence rate on the database level.
* `support_instant_and_daily_backups` - If 'true', daily and instant backups are supported.
* `support_replication` - (Optional) Databases replication support. Either 'true' or 'false'.
* `support_clustering` - Databases clustering support. Either 'true' or 'false'.
* `supported_alerts` - A list of the alerts supported by databases created under Subscriptions from this Plan.
* `customer_support` - The level of customer support available. E.g., 'Basic', 'Standard'.
