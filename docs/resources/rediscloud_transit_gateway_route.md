---
page_title: "Redis Cloud: rediscloud_transit_gateway_route"
description: |-
  Transit Gateway Route resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_transit_gateway_route

Manages Transit Gateway routing (CIDRs) for a Pro subscription in your Redis Enterprise Cloud Account.

This resource should be used after the Transit Gateway attachment has been accepted on both the Redis Cloud and AWS sides. Use `depends_on` to ensure proper ordering.

## Example Usage

```hcl
# Accept the TGW invitation (Redis Cloud side)
resource "rediscloud_transit_gateway_invitation_acceptor" "example" {
  subscription_id   = rediscloud_subscription.example.id
  tgw_invitation_id = data.rediscloud_transit_gateway_invitations.example.invitations[0].id
  action            = "accept"
}

# Create the attachment
resource "rediscloud_transit_gateway_attachment" "example" {
  subscription_id = rediscloud_subscription.example.id
  tgw_id          = data.rediscloud_transit_gateway.example.tgw_id
}

# Accept on AWS side
resource "aws_ec2_transit_gateway_vpc_attachment_accepter" "example" {
  transit_gateway_attachment_id = rediscloud_transit_gateway_attachment.example.attachment_uid
}

# Configure CIDRs (depends on AWS acceptance)
resource "rediscloud_transit_gateway_route" "example" {
  subscription_id = rediscloud_subscription.example.id
  tgw_id          = data.rediscloud_transit_gateway.example.tgw_id
  cidrs           = ["10.10.20.0/24", "10.10.30.0/24"]

  depends_on = [aws_ec2_transit_gateway_vpc_attachment_accepter.example]
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro subscription
* `tgw_id` - (Required) The ID of the Transit Gateway
* `cidrs` - (Required) A list of consumer CIDR blocks

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 5 mins) Used when creating the route
* `read` - (Defaults to 3 mins) Used when reading the route
* `update` - (Defaults to 5 mins) Used when updating the route
* `delete` - (Defaults to 5 mins) Used when destroying the route

## Attribute Reference

* `id` - The identifier of the route, in the format `{subscription_id}/{tgw_id}`

## Import

`rediscloud_transit_gateway_route` can be imported using the ID of the Pro subscription and the ID of the Transit Gateway in the format {subscription_id}/{tgw_id}, e.g.

```
$ terraform import rediscloud_transit_gateway_route.example 123456/47
```
