---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_essentials_subscription"
description: |-
  Essentials Subscription data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_essentials_subscription

This data source allows access to the details of an existing subscription within your Redis Enterprise Cloud account.

-> **Note:** This is referring to Essentials Subscriptions only. See also `rediscloud_subscription` (Pro) and `rediscloud_active_active_subscription`. 

## Example Usage

The following example shows how to use the name attribute to locate an essentials subscription within your Redis Enterprise Cloud account.

```hcl
data "rediscloud_essentials_subscription" "example" {
  name = "My Example Subscription"
}
output "rediscloud_essentials_subscription" {
  value = data.rediscloud_essentials_subscription.example.id
}
```

## Argument Reference

* `id` - (Optional) The subscription's id
* `name` - (Optional) A convenient name for the plan.

## Attributes Reference

* `status` - The current status of the subscription
* `plan_id` - The plan to which this subscription belongs
* `payment_method_id` - A valid payment method pre-defined in the current account
* `creation_date` - When the subscription was created
