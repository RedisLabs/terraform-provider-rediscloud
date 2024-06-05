---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_payment_method"
description: |-
  Payment method data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_payment_method

The Payment Method data source allows access to the ID of a Payment Method configured against your Redis Enterprise Cloud account. This ID can be used when creating Subscription resources.

## Example Usage

The following example shows a payment method data source being used with a subscription resource.  

The example assumes only a single payment method has been defined with a card type of Visa.  By default all expired payment methods are excluded.

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "example" {

  name = "My Test Subscription"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  ...
}
```

The following example shows how a single payment method can be identified when there are multiple methods against the same card type.

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
  last_four_numbers = "0123"
}
```

## Argument Reference

* `card_type` - (Optional) Type of card that the payment method should be, such as `Visa`.

* `last_four_numbers` - (Optional) Last four numbers of the card of the payment method.

* `exclude_expired` - (Optional) Whether to exclude any expired cards or not. Default is `true`.

## Attributes Reference

`id` is set to the ID of the found payment method.
