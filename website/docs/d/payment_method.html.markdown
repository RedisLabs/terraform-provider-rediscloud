---
layout: "rediscloud"
page_title: "RedisCloud: rediscloud_payment_method"
sidebar_current: "docs-rediscloud-payment-method"
description: |-
  Payment method data source in the Terraform provider RedisCloud.
---

# rediscloud_payment_method

Use this data source to get the ID of a payment method for use with the subscription resource.

## Example Usage

```hcl
data "rediscloud_payment_method" "example" {
}
```

## Argument Reference

* `card_type` - (Optional) Type of card that the payment method should be, such as `Visa`.

* `last_four_numbers` - (Optional) Last four numbers of the card of the payment method.

## Attributes Reference

`id` is set to the ID of the found payment method.
