---
page_title: "Redis Cloud: rediscloud_active_active_transit_gateway_route"
description: |-
  Active-Active Transit Gateway Route resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_transit_gateway_route

Manages Transit Gateway routing (CIDRs) for an Active-Active Subscription in your Redis Enterprise Cloud Account.

This resource should be used after the Transit Gateway attachment has been accepted on both the Redis Cloud and AWS sides. Use `depends_on` to ensure proper ordering.

## Example Usage

```hcl
# Accept the TGW invitation (Redis Cloud side)
resource "rediscloud_active_active_transit_gateway_invitation_acceptor" "example" {
  subscription_id   = rediscloud_active_active_subscription.example.id
  region_id         = 1
  tgw_invitation_id = data.rediscloud_active_active_transit_gateway_invitations.example.invitations[0].id
  action            = "accept"
}

# Create the attachment
resource "rediscloud_active_active_transit_gateway_attachment" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  region_id       = 1
  tgw_id          = data.rediscloud_active_active_transit_gateway.example.tgw_id
}

# Accept on AWS side
resource "aws_ec2_transit_gateway_vpc_attachment_accepter" "example" {
  transit_gateway_attachment_id = rediscloud_active_active_transit_gateway_attachment.example.attachment_uid
}

# Configure CIDRs (depends on AWS acceptance)
resource "rediscloud_active_active_transit_gateway_route" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  region_id       = 1
  tgw_id          = data.rediscloud_active_active_transit_gateway.example.tgw_id
  cidrs           = ["10.10.20.0/24", "10.10.30.0/24"]

  depends_on = [aws_ec2_transit_gateway_vpc_attachment_accepter.example]
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Active-Active subscription
* `region_id` - (Required) The ID of the AWS region
* `tgw_id` - (Required) The ID of the Transit Gateway
* `cidrs` - (Required) A list of consumer CIDR blocks

## Attribute Reference

No additional attributes are exported.

## Import

`rediscloud_active_active_transit_gateway_route` can be imported using the ID of the Active-Active subscription, region ID, and Transit Gateway ID in the format {subscription_id}/{region_id}/{tgw_id}, e.g.

```
$ terraform import rediscloud_active_active_transit_gateway_route.example 123456/1/47
```
