---
page_title: "Redis Cloud: rediscloud_active_active_subscription"
description: |-
  Active Active Subscription data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_active_active_subscription

This data source allows access to the details of an existing subscription within your Redis Enterprise Cloud account.

-> **Note:** This is referring to Active-Active Subscriptions only. See also `rediscloud_subscription` (Pro) and `rediscloud_essentials_subscription`.

## Example Usage

The following example shows how to use the name attribute to locate a subscription within your Redis Enterprise
Cloud account.

```hcl
data "rediscloud_active_active_subscription" "example" {
  name = "My AA Subscription"
}

output "rediscloud_active_active_subscription" {
  value = data.rediscloud_active_active_subscription.example.id
}
```

## Argument Reference

* `name` - (Required) The name of the subscription to filter returned subscriptions

## Attribute reference

`id` is set to the ID of the found subscription.

* `aws_account_id` - AWS account ID that the subscription is deployed in (AWS subscriptions only).
* `payment_method` (Optional) The payment method for the requested subscription, (either `credit-card`
  or `marketplace`). If `credit-card` is specified, `payment_method_id` must be defined. Default: 'credit-card'. **(
  Changes to) this attribute are ignored after creation.**
* `payment_method_id` - A valid payment method pre-defined in the current account
* `cloud_provider` - The cloud provider used with the subscription, (either `AWS` or `GCP`).
* `number_of_databases` - The number of databases that are linked to this subscription.
* `status` - Current status of the subscription
* `customer_managed_key_enabled` - Whether customer managed key encryption is enabled for the subscription
* `customer_managed_key_deletion_grace_period` - The deletion grace period for the customer managed key (e.g. 'immediate', '15-minutes')
* `customer_managed_key_redis_service_account` - The Redis service account principal associated with the subscription. This is used to grant access to the customer managed encryption key (GCP subscriptions).
* `customer_managed_key_aws_role_arn` - The ARN of the IAM role used by the subscription to access the AWS KMS customer managed key. Grant this role access to your KMS key via key policy (AWS subscriptions).
* `public_endpoint_access` - Whether public endpoint access is enabled for databases in the subscription
* `maintenance_windows` - Details about the subscription's maintenance window specification, documented below
* `pricing` - A list of pricing objects, documented below
* `resource_tags` - A string/string map of tags to assign to the cloud resources created by this subscription

The `maintenance_windows` object has these attributes:

* `mode` - Either `automatic` (Redis specified) or `manual` (User specified)
* `window` - A list of windows (if manual mode)

The `window` object has these attributes:

* `start_hour` - What hour in the day (0-23) the window opens
* `duration_in_hours` - How long the window is open
* `days` - A list of weekdays on which the window is open ('Monday', 'Tuesday' etc)

The `pricing` object has these attributes:

* `database_name` - The database this pricing entry applies to.
* `type` - The type of cost e.g. 'Shards'.
* `typeDetails` - Further detail e.g. 'micro'.
* `quantity` - The number of units this pricing entry covers.
* `quantityMeasurement` - The unit that `quantity` is measured in, e.g. 'shards'.
* `pricePerUnit` - The cost of a single unit.
* `priceCurrency` - The currency the price is denominated in, e.g. 'USD'.
* `pricePeriod` - The billing period the price applies to, e.g. 'hour'.
* `region` - The region this cost is associated with, if any.
