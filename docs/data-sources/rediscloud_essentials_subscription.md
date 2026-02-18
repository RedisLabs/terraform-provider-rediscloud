---
page_title: "Redis Cloud: rediscloud_essentials_subscription"
description: |-
  Essentials Subscription data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_essentials_subscription

This data source allows access to the details of an existing Essentials subscription within your Redis Enterprise Cloud account.

-> **Note:** This is referring to Essentials subscriptions only. See also `rediscloud_subscription` (Pro) and `rediscloud_active_active_subscription`.

## Example Usage

The following example shows how to use the name attribute to locate an Essentials subscription within your Redis Enterprise Cloud account.

```hcl
data "rediscloud_essentials_subscription" "example" {
  name = "My Example Subscription"
}
output "rediscloud_essentials_subscription" {
  value = data.rediscloud_essentials_subscription.example.id
}
```

## Argument Reference

* `subscription_id` - (Optional) The ID of the Essentials subscription to look up.
* `name` - (Optional) A meaningful name to identify the subscription.

## Attributes Reference

* `id` - The ID of the Essentials subscription
* `subscription_id` - The ID of the Essentials subscription
* `name` - The name of the subscription
* `status` - The current status of the subscription
* `plan_id` - The ID of the plan to which this subscription belongs
* `payment_method_id` - The ID of the payment method pre-defined in the current account
* `creation_date` - When the subscription was created
