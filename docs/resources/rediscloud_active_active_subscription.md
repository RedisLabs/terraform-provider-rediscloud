---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_subscription"
description: |-
  Subscription resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_subscription

Creates an Active-Active Subscription within your Redis Enterprise Cloud Account.
This resource is responsible for creating and managing subscriptions.

~> **Note:** The payment_method property is ignored after Subscription creation.

~> **Note:** The creation_plan block allows the API server to create a well-optimised infrastructure for your databases in the cluster.
The attributes inside the block are used by the provider to create initial
databases. Those databases will be deleted after provisioning a new
subscription, then the databases defined as separate resources will be attached to
the subscription. The creation_plan block can ONLY be used for provisioning new
subscriptions, the block will be ignored if you make any further changes or try importing the resource (e.g. `terraform import` ...).

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
    dataset_size_in_gb = 1
    quantity = 1
    modules = ["RedisJSON"]
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
      read_operations_per_second = 1000
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A meaningful name to identify the subscription
* `payment_method` (Optional) The payment method for the requested subscription, (either `credit-card` or `marketplace`). If `credit-card` is specified, `payment_method_id` must be defined. Default: 'credit-card'. **(Changes to) this attribute are ignored after creation.**
* `payment_method_id` - (Optional) A valid payment method pre-defined in the current account. This value is __Optional__ for AWS/GCP Marketplace accounts, but __Required__ for all other account types
* `cloud_provider` - (Optional) The cloud provider to use with the subscription, (either `AWS` or `GCP`). Default: ‘AWS’. **Modifying this attribute will force creation of a new resource.**
* `redis_version` - (Optional) The Redis version of the databases in the subscription. If omitted, the Redis version will be the default. **Modifying this attribute will force creation of a new resource.**
* `creation_plan` - (Required) A creation plan object, documented below. Ignored after creation.
* `maintenance_windows` - (Optional) The subscription's maintenance window specification, documented below.

The `creation_plan` block supports:

* `memory_limit_in_gb` - (Optional -  **Required if `dataset_size_in_gb` is unset**) Maximum memory usage for this specific database, including replication and other overhead **Deprecated in favor of `dataset_size_in_gb` - not possible to import databases with this attribute set**
* `dataset_size_in_gb` - (Optional - **Required if `memory_limit_in_gb` is unset**) The maximum amount of data in the dataset for this specific database is in GB
* `quantity` - (Required) The planned number of databases in the subscription.
* `modules` - (Optional) A list of modules to be enabled on all deployments of this database. Either: `RedisJSON` or `RediSearch`.
* `region` - (Required) Deployment region block, documented below

The creation_plan `region` block supports:

* `region` - (Required) Deployment region as defined by the cloud provider
* `networking_deployment_cidr` - (Required) Deployment CIDR mask. The total number of bits must be 24 (x.x.x.x/24)
* `write_operations_per_second` - (Required) Throughput measurement for an active-active subscription
* `read_operations_per_second` - (Required) Throughput measurement for an active-active subscription

The `maintenance_windows` object has these attributes:

* `mode` - Either `automatic` (Redis specified) or `manual` (User specified)
* `window` - A list of windows (if manual mode)

The `window` object has these attributes:

* `start_hour` - What hour in the day (0-23) the window opens
* `duration_in_hours` - How long the window is open
* `days` - A list of weekdays on which the window is open ('Monday', 'Tuesday' etc)

~> **Note:** If changes are made to attributes in the subscription which require the subscription to be recreated (such as `cloud_provider`), the creation_plan will need to be defined in order to change these attributes. This is because the creation_plan is always required when a subscription is created.

## Attribute reference

* `pricing` - A list of pricing objects, documented below

The `pricing` object has these attributes:

* `type` - The type of cost. E.g. 'Shards'.
* `typeDetails` - Further detail E.g. 'micro'.
* `quantity` - Self-explanatory.
* `quantityMeasurement` - Self-explanatory.
* `pricePerUnit` - Price per Unit.
* `priceCurrency` - The price currency
* `pricePeriod` - Price period. E.g. 'hour'.
* `region` - Specify if the cost is associated with a particular region.

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 mins) Used when creating the subscription
* `update` - (Defaults to 30 mins) Used when updating the subscription
* `delete` - (Defaults to 10 mins) Used when destroying the subscription

## Import

`rediscloud_active_active_subscription` can be imported using the ID of the subscription, e.g.

```
$ terraform import rediscloud_active_active_subscription.subscription-resource 12345678
```

~> **Note:** the payment_method property and creation_plan block will be ignored during imports.

~> **Note:** when importing an existing Subscription, upon providing a `redis_version`, Terraform will always try to
recreate the resource. The API doesn't return this value, so we can't detect changes between states.
