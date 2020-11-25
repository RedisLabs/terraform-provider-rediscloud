---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription_peering"
description: |-
  Subscription VPC peering resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_subscription_peering

Creates an AWS or GCP VPC peering for an existing Redis Enterprise Cloud Subscription, allowing access to your subscription databases as if they were on the same network.

For AWS, peering should be accepted by the other side.
For GCP, the opposite peering request should be submitted.

## Example Usage - AWS

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

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 10 mins) Used when creating the peering connection
* `delete` - (Defaults to 10 mins) Used when deleting the peering connection

## Attribute reference

* `status` is set to the current status of the account - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`.

## Import

`rediscloud_subscription_peering` can be imported using the ID of the subscription and the ID of the peering connection, e.g.

```
$ terraform import rediscloud_subscription_peering.example 12345678/1234
```
