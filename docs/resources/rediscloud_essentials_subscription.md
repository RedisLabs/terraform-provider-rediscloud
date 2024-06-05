---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_essentials_subscription"
description: |-
  Essentials Subscription resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_essentials_subscription

Creates an Essentials Subscription within your Redis Enterprise Cloud Account.

~> **Note:** It's recommended to create a Database (`rediscloud_essentials_database`) when you create a Subscription.

## Example Usage

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_essentials_plan" "plan" {
  name = "Single-Zone_1GB"
  cloud_provider = "AWS"
  region = "us-west-1"
}

resource "rediscloud_essentials_subscription" "subscription-resource" {
  name              = "subscription-name"
  plan_id = data.rediscloud_essentials_plan.plan.id
  payment_method_id = data.rediscloud_payment_method.card.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A meaningful name to identify the subscription
* `plan_id` - (Required) The plan to which this subscription will belong
* `payment_method_id` - (Optional) If the plan is paid, this must be a valid payment method pre-defined in the current account

### Timeouts

The `timeouts` block allows you to
specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 mins) Used when creating the subscription
* `update` - (Defaults to 30 mins) Used when updating the subscription
* `delete` - (Defaults to 10 mins) Used when destroying the subscription

## Attribute reference

* `status` - The current status of the subscription
* `creation_date` - When the subscription was created

## Import

`rediscloud_essentials_subscription` can be imported using the ID of the subscription, e.g.

```
$ terraform import rediscloud_essentials_subscription.subscription-resource 12345678
```
