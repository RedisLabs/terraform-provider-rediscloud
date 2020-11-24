---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_payment_method"
sidebar_current: "docs-rediscloud-payment-method"
description: |-
  Payment method data source in the Terraform provider Redis Cloud.
---

# Data Source: rediscloud_payment_method

Use this data source to get the ID of a payment method for use with the subscription resource.

## Example Usage

```hcl
data "rediscloud_payment_method" "example" {
}
```

## Argument Reference

* `card_type` - (Optional) Type of card that the payment method should be, such as `Visa`.

* `last_four_numbers` - (Optional) Last four numbers of the card of the payment method.

* `exclude_expired` - (Optional) Whether to exclude any expired cards or not. Default is `true`.

## Attributes Reference

`id` is set to the ID of the found payment method.
