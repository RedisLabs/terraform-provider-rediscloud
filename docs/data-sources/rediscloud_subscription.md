---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription"
description: |-
  Pro Subscription data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_subscription

This data source allows access to the details of an existing Subscription within your Redis Enterprise Cloud account.

-> **Note:** This is referring to Pro Subscriptions only. See also `rediscloud_active_active_subscription` and `rediscloud_essentials_subscription`.

## Example Usage

The following example shows how to use the name attribute to locate a subscription within your Redis Enterprise Cloud account.

```hcl
data "rediscloud_subscription" "example" {
  name = "My Example Subscription"
}
output "rediscloud_subscription" {
  value = data.rediscloud_subscription.example.id
}
```

## Argument Reference

* `name` - (Optional) The name of the subscription to filter returned subscriptions

## Attributes Reference

`id` is set to the ID of the found subscription.

* `payment_method_id` - A valid payment method pre-defined in the current account
* `memory_storage` - Memory storage preference: either ‘ram’ or a combination of 'ram-and-flash’
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
* `networking_vpc_id` - The ID of the VPC where the Redis Cloud subscription is deployed.
* `preferred_availability_zones` - List of availability zones used

* `networks` - List of generated network configuration

The `networks` block has these attributes:

* `networking_subnet_id` - The subnet that the subscription deploys into
* `networking_deployment_cidr` - Deployment CIDR mask for the generated
* `networking_vpc_id` - VPC id for the generated network
