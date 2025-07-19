---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_dedicated_subscription"
description: |-
  Dedicated subscription resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_dedicated_subscription

Creates a Dedicated Subscription within your Redis Enterprise Cloud Account. Dedicated subscriptions provide dedicated infrastructure with guaranteed resources and enhanced performance for enterprise workloads.

-> **Note:** This resource is for Dedicated Subscriptions only. See also `rediscloud_subscription` (Pro), `rediscloud_essentials_subscription`, and `rediscloud_active_active_subscription`.

~> **Warning:** Dedicated subscriptions are currently in development and may not be available in all environments. Contact Redis support for availability.

## Example Usage

### Basic Dedicated Subscription

```hcl
resource "rediscloud_dedicated_subscription" "example" {
  name           = "my-dedicated-subscription"
  payment_method = "credit-card"

  cloud_provider {
    provider                     = "AWS"
    cloud_account_id            = "1"
    region                      = "us-east-1"
    networking_deployment_cidr  = "10.0.0.0/24"
    availability_zones          = ["us-east-1a", "us-east-1b"]
  }

  instance_type {
    instance_name = "dedicated-large"
    replication   = true
  }

  redis_version = "7.2"
}
```

### Dedicated Subscription with Custom VPC

```hcl
resource "rediscloud_dedicated_subscription" "custom_vpc" {
  name           = "dedicated-custom-vpc"
  payment_method = "credit-card"

  cloud_provider {
    provider                     = "AWS"
    cloud_account_id            = "1"
    region                      = "us-west-2"
    networking_deployment_cidr  = "172.16.0.0/24"
    networking_vpc_id           = "vpc-12345678"
    availability_zones          = ["us-west-2a", "us-west-2b", "us-west-2c"]
  }

  instance_type {
    instance_name = "dedicated-xlarge"
    replication   = true
  }
}
```

### GCP Dedicated Subscription

```hcl
resource "rediscloud_dedicated_subscription" "gcp_example" {
  name           = "gcp-dedicated-subscription"
  payment_method = "credit-card"

  cloud_provider {
    provider                     = "GCP"
    cloud_account_id            = "1"
    region                      = "us-central1"
    networking_deployment_cidr  = "10.1.0.0/24"
    availability_zones          = ["us-central1-a", "us-central1-b"]
  }

  instance_type {
    instance_name = "dedicated-medium"
    replication   = false
  }

  redis_version = "6.2"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) A meaningful name to identify the dedicated subscription.
* `payment_method` - (Optional) Payment method for the requested subscription. Either `credit-card` or `marketplace`. Default: `credit-card`.
* `payment_method_id` - (Optional) A valid payment method pre-defined in the current account. Required when `payment_method` is `credit-card`.
* `cloud_provider` - (Required, change forces recreation) Configuration for cloud provider settings. See [Cloud Provider](#cloud-provider) below.
* `instance_type` - (Required, change forces recreation) Dedicated instance type specification. See [Instance Type](#instance-type) below.
* `redis_version` - (Optional, change forces recreation) Version of Redis to create.

### Cloud Provider

The `cloud_provider` block supports:

* `provider` - (Optional, change forces recreation) The cloud provider to use with the subscription. Either `AWS` or `GCP`. Default: `AWS`.
* `cloud_account_id` - (Optional, change forces recreation) Cloud account identifier. Default: Redis Labs internal cloud account (`1`).
* `region` - (Required, change forces recreation) Deployment region as defined by cloud provider.
* `availability_zones` - (Optional, change forces recreation) List of availability zones used.
* `networking_deployment_cidr` - (Required, change forces recreation) Deployment CIDR mask.
* `networking_vpc_id` - (Optional, change forces recreation) Either an existing VPC Id (already exists in the specific region) or create a new VPC (if no VPC is specified).

### Instance Type

The `instance_type` block supports:

* `instance_name` - (Required, change forces recreation) The name of the dedicated instance type. Available instance types can be retrieved from the instance-types endpoint.
* `replication` - (Optional, change forces recreation) Enable replication for high availability. Default: `true`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the dedicated subscription.
* `status` - Current status of the subscription.
* `pricing` - Pricing details for this dedicated subscription. See [Pricing](#pricing) below.

### Pricing

The `pricing` block contains:

* `type` - The type of cost (e.g., 'Instance').
* `type_details` - Further detail (e.g., instance type name).
* `quantity` - Number of instances.
* `quantity_measurement` - Unit of measurement.
* `price_per_unit` - Price per unit.
* `price_currency` - Currency (e.g., 'USD').
* `price_period` - Billing period (e.g., 'hour').
* `region` - Region associated with the cost.

## Timeouts

`rediscloud_dedicated_subscription` provides the following [Timeouts](https://www.terraform.io/docs/configuration/blocks/resources/syntax.html#operation-timeouts) configuration options:

* `create` - (Default `30 minutes`) Used for creating dedicated subscriptions.
* `read` - (Default `10 minutes`) Used for reading dedicated subscriptions.
* `update` - (Default `30 minutes`) Used for updating dedicated subscriptions.
* `delete` - (Default `10 minutes`) Used for deleting dedicated subscriptions.

## Import

`rediscloud_dedicated_subscription` can be imported using the subscription ID, e.g.

```
$ terraform import rediscloud_dedicated_subscription.example 12345
```

~> **Note:** When importing dedicated subscriptions, ensure that the configuration matches the existing subscription settings to avoid unexpected changes.
