---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_transit_gateway_attachment"
description: |-
  Active-Active Transit Gateway Attachment resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_transit_gateway_attachment

Manages a Transit Gateway Attachment to an Active-Active Subscription in your Redis Enterprise Cloud Account.

## Example Usage

```hcl
data "rediscloud_active_active_transit_gateway" "gateway" {
  subscription_id = "113492"
  region_id       = 1
  aws_tgw_id      = "tgw-1c55bfdoe20pdsad2"
}

resource "rediscloud_active_active_transit_gateway_attachment" "attachment" {
  subscription_id = "113492"
  region_id       = 1
  tgw_id          = data.rediscloud_active_active_transit_gateway.gateway.tgw_id
}

# Use rediscloud_active_active_transit_gateway_route to configure CIDRs
resource "rediscloud_active_active_transit_gateway_route" "route" {
  subscription_id = "113492"
  region_id       = 1
  tgw_id          = data.rediscloud_active_active_transit_gateway.gateway.tgw_id
  cidrs           = ["10.10.20.0/24"]

  depends_on = [aws_ec2_transit_gateway_vpc_attachment_accepter.example]
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Active-Active subscription to attach
* `region_id` - (Required) The ID of the AWS region
* `tgw_id` - (Required) The ID of the Transit Gateway to attach to
* `cidrs` - (Optional) A list of consumer CIDR blocks. It is recommended to use the [`rediscloud_active_active_transit_gateway_route`](rediscloud_active_active_transit_gateway_route.md) resource instead for managing CIDRs.

## Attribute Reference

* `aws_tgw_uid` - The ID of the Transit Gateway as known to AWS
* `attachment_uid` - A unique identifier for the Subscription/Transit Gateway attachment, if established
* `status` - The status of the Transit Gateway
* `attachment_status` - The status of the Subscription/Transit Gateway attachment, if established
* `aws_account_id` - The Transit Gateway's AWS account ID
* `cidrs` - A list of consumer CIDR blocks

## Import

`rediscloud_active_active_transit_gateway_attachment` can be imported using the ID of the Active-Active subscription, region ID, and Transit Gateway ID in the format {subscription_id}/{region_id}/{tgw_id}, e.g.

```
$ terraform import rediscloud_active_active_transit_gateway_attachment.tgwa-resource 123456/1/47
```
