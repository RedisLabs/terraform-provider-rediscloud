---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_transit_gateway_invitation_acceptor"
description: |-
  Transit Gateway Invitation Acceptor resource for a Pro Subscription in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_transit_gateway_invitation_acceptor

Manages the acceptance or rejection of a Transit Gateway invitation in your Redis Enterprise Cloud Account. Transit Gateway invitations are created when an AWS Resource Share shares a Transit Gateway with your subscription.

## Example Usage

```hcl
data "rediscloud_transit_gateway_invitations" "example" {
  subscription_id = rediscloud_subscription.example.id
}

resource "rediscloud_transit_gateway_invitation_acceptor" "example" {
  subscription_id   = rediscloud_subscription.example.id
  tgw_invitation_id = data.rediscloud_transit_gateway_invitations.example.invitations[0].id
  action            = "accept"
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro subscription. **Modifying this attribute will force creation of a new resource.**
* `tgw_invitation_id` - (Required) The ID of the Transit Gateway invitation to accept or reject. **Modifying this attribute will force creation of a new resource.**
* `action` - (Required) Accept or reject the invitation. Accepted values are `accept` and `reject`.

## Attribute Reference

* `tgw_id` - The Transit Gateway ID relative to the associated subscription
* `aws_tgw_uid` - The AWS Transit Gateway ID
* `status` - The status of the invitation
* `aws_account_id` - The AWS account ID associated with the Transit Gateway

## Import

`rediscloud_transit_gateway_invitation_acceptor` can be imported using the format `{subscription_id}/{tgw_invitation_id}`, e.g.

```
$ terraform import rediscloud_transit_gateway_invitation_acceptor.example 123456/7890
```

Note: The `action` attribute is not stored in the API and will not be populated during import.
