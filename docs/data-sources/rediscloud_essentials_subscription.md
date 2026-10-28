---
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
* `name` - (Optional) A meaningful name to identify the subscription

## Attributes Reference

* `status` - The status of this subscription
* `plan_id` - The identifier of the plan to template the subscription
* `payment_method_id` - The identifier of the method which will be charged for this subscription. Not required for free plans
* `creation_date` - The date/time this subscription was created
