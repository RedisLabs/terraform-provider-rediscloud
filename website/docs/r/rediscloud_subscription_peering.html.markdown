---
layout: "rediscloud"
page_title: "RedisCloud: rediscloud_subscription_peering"
sidebar_current: "docs-rediscloud-subscription-peering"
description: |-
  Subscription VPC peering resource in the Terraform provider RedisCloud.
---

# rediscloud_subscription_peering

Subscription VPC peering resource in the Terraform provider RedisCloud.

## Example Usage

```hcl
resource "rediscloud_subscription" "example" {
  // ...
}

resource "rediscloud_subscription_peering" "example" {
   subscription_id = rediscloud_subscription.example.id
   region = "eu-west-1"
   aws_account_id = "123456789012"
   vpc_id = "vpc-01234567890"
   vpc_cidr = "10.0.0.0/8"
}
```

## Argument Reference

The following arguments are supported:

* `subscription_id` - (Required) A valid subscription predefined in the current account
* `region` - (Required) AWS Region that the VPC to be peered lives in
* `aws_account_id` - (Required) AWS account id that the VPC to be peered lives in
* `vpc_id` - (Required) Identifier of the VPC to be peered
* `vpc_cidr` - (Required) CIDR range of the VPC to be peered 

## Attribute reference

`status` is set to the current status of the account - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`.
