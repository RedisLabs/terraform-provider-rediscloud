---
layout: "rediscloud"
page_title: "RedisCloud: rediscloud_subscription"
sidebar_current: "docs-rediscloud-subscription"
description: |-
  Subscription resource in the Terraform provider RedisCloud.
---

# rediscloud_subscription

Subscription resource in the Terraform provider RedisCloud.

## Example Usage

```hcl
resource "rediscloud_subscription" "example" {
  sample_attribute = "foo"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) A meaningful name to identify the subscription
* `dryRun` - (Optional) When 'false’: Creates a deployment plan and deploys it (creating any resources required by the plan). When 'true’: creates a read-only deployment plan without any resource creation. Default: ‘true’
* `payment_method_id` - (Required) A valid payment method pre-defined in the current account
* `memory_storage` - (Optional) Memory storage preference: either ‘ram’ or a combination of 'ram-and-flash’. Default: ‘ram’
* `persistent_storage_encryption` - (Optional) Encrypt data stored in persistent storage. Required for a GCP subscription. Default: ‘false’
* `cloud_providers` - (Required) 
* `databases` - (Required)

The `cloud_providers` block supports:

* `provider` - (Optional) The cloud provider to use with the subscription, (either `AWS` or `GCP`). Default: ‘AWS’
* `cloud_account_id` - (Optional) Cloud account identifier. Default: Redis Labs internal cloud account (using Cloud Account Id = 1 implies using Redis Labs internal cloud account). Note that a GCP subscription can be created only with Redis Labs internal cloud account.
* `regions` - (Required) Cloud networking details, per region (single region or multiple regions for Active-Active cluster only) 

The `databases` block supports:

* `name` - (Required) A meaningful name to identify the database
* `protocol` - (Optional) The protocol that will be used to access the database, (either ‘redis’ or 'memcached’) Default: ‘redis’
* `memory_limit_in_gb` - (Required) Maximum memory usage for this specific database
* `support_oss_cluster_api` - (Optional) Support Redis open-source (OSS) Cluster API. Default: ‘false’
* `data_persistence` - (Optional) Rate of database data persistence (in persistent storage). Default: ‘none’
* `replication` - (Optional) Databases replication. Default: ‘true’
* `throughput_measurement_by` - (Required) Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)
* `throughput_measurement_value` - (Required) Throughput value (as applies to selected measurement method)
* `modules` - (Optional) Redis Labs modules to be provisioned in the database
* `quantity` - (Optional) Number of databases (of this type) that will be created. Default: 1
* `average_item_size_in_bytes` - (Optional) Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes) of the items stored in the database. Default: 1000

The cloud_provider region block supports:

* `region` - (Required) Deployment region as defined by cloud provider
* `multiple_availability_zones` - (Optional) Support deployment on multiple availability zones within the selected region. Default: ‘false’
* `preferred_availability_zones` - (Optional) Availability zones deployment preferences (for the selected provider & region). Example = '["us-east-1a", "us-east-1c", "us-east-2e"]'
* `networking_deployment_cidr` - (Required) Deployment CIDR mask. Default: If using Redis Labs internal cloud account, 192.168.0.0/24
* `networking_vpc_id` - (Optional) Either an existing VPC Id (already exists in the specific region) or create a new VPC (if no VPC is specified). VPC Identifier must be in a valid format (for example: ‘vpc-0125be68a4625884ad’) and existing within the hosting account