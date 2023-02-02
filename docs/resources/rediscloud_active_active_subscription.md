---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_subscription"
description: |-
  Subscription resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_active_active_subscription

Creates an Active Active Subscription within your Redis Enterprise Cloud Account.
This resource is responsible for creating subscriptions and the databases within
that subscription. This allows Redis Enterprise Cloud to provision
your databases defined in separate resources in the most efficient way.

~> **Note:** The creation_plan block allows the API server to create a well-optimised hardware specification for your databases in the cluster.
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
  
resource "rediscloud_active_active_subscription" "example" {
	name = "example"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "AWS"
   
	creation_plan {
	  memory_limit_in_gb = 1
	  quantity = 1
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
* `payment_method` (Optional) The payment method for the requested subscription, (either `credit-card` or `marketplace`). If `credit-card` is specified, `payment_method_id` must be defined.
* `payment_method_id` - (Optional) A valid payment method pre-defined in the current account. This value is __Optional__ for AWS/GCP Marketplace accounts, but __Required__ for all other account types. 
* `cloud_provider` - (Required) A cloud provider object, documented below 
* `creation_plan` - (Required) A creation plan object, documented below

The `cloud_provider` block supports:

* `provider` - (Optional) The cloud provider to use with the subscription, (either `AWS` or `GCP`). Default: ‘AWS’
* `cloud_account_id` - (Optional) Cloud account identifier. Default: Redis Labs internal cloud account
(using Cloud Account ID = 1 implies using Redis Labs internal cloud account). Note that a GCP subscription can be created
only with Redis Labs internal cloud account
* `region` - (Required) Cloud networking details, per region, documented below

The `creation_plan` block supports:

* `memory_limit_in_gb` - (Required) Maximum memory usage that will be used for your largest planned database.
* `quantity` - (Required) The planned number of databases in the subscription.


~> **Note:** If changes are made to attributes in the subscription which require the subscription to be recreated (such as `memory_storage`, `cloud_provider` or `payment_method`), the creation_plan will need to be defined in order to change these attributes. This is because the creation_plan is always required when a subscription is created.

The cloud_provider `region` block supports:

* `region` - (Required) Deployment region as defined by cloud provider
* `networking_deployment_cidr` - (Required) Deployment CIDR mask.
* `write_operations_per_second` - (Required) Throughput measurement for an active-active subscription
* `read_operations_per_second` - (Required) Throughput measurement for an active-active subscription


### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 mins) Used when creating the subscription
* `update` - (Defaults to 30 mins) Used when updating the subscription
* `delete` - (Defaults to 10 mins) Used when destroying the subscription

## Import

`rediscloud_active_active_subscription` can be imported using the ID of the subscription, e.g.

```
$ terraform import rediscloud_subscription.example 12345678
```

~> **Note:** the creation_plan block will be ignored during imports.
