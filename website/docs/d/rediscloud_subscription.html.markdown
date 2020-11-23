---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription"
sidebar_current: "docs-rediscloud-subscription"
description: |-
  Subscription data source in the Terraform provider Redis Cloud.
---

# Data Source: rediscloud_subscription

Use this data source to get the details of an existing subscription within your Redis Cloud account.

## Example Usage

```hcl
data "rediscloud_subscription" "example" {
}
```

## Argument Reference

* `name` - (Optional) The name of the subscription to filter returned subscriptions

## Attributes Reference

`id` is set to the ID of the found subscription.

* `payment_method_id` - A valid payment method pre-defined in the current account
* `memory_storage` - Memory storage preference: either ‘ram’ or a combination of 'ram-and-flash’
* `persistent_storage_encryption` - Encrypt data stored in persistent storage. Required for a GCP subscription.
* `cloud_provider` - A cloud provider object, documented below
* `number_of_databases` - The number of databases that are linked to this subscription.
* `status` - Current status of the subscription 

The `cloud_provider` block supports:

* `provider` - The cloud provider to use with the subscription, (either `AWS` or `GCP`)
* `cloud_account_id` - Cloud account identifier, (A Cloud Account Id = 1 implies using Redis Labs internal cloud account)
* `region` - Cloud networking details, per region (single region or multiple regions for Active-Active cluster only), documented below

The cloud_provider `region` block supports:

* `region` - Deployment region as defined by cloud provider
* `multiple_availability_zones` - Support deployment on multiple availability zones within the selected region
* `networking_deployment_cidr` - Deployment CIDR mask
* `networking_subnet_id` - The subnet that the subscription deploys into
* `networking_vpc_id` - The ID of the VPC where the Redis Cloud subscription is deployed.
* `preferred_availability_zones` - List of availability zones used
