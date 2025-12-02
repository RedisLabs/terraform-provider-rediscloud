---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_transit_gateway_invitations"
description: |-
  Transit Gateway Invitations data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_transit_gateway_invitations

The Transit Gateway Invitations data source allows access to pending Transit Gateway invitations within your Redis Enterprise Cloud Subscription. These invitations are created when an AWS Resource Share shares a Transit Gateway with your subscription.

## Example Usage

```hcl
data "rediscloud_transit_gateway_invitations" "example" {
  subscription_id = "113991"
}

output "pending_invitations" {
  value = data.rediscloud_transit_gateway_invitations.example.invitations
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of a Pro subscription

## Attribute Reference

* `invitations` - List of Transit Gateway invitations, documented below

The `invitations` object has these attributes:

* `id` - The ID of the Transit Gateway invitation
* `tgw_id` - The Transit Gateway ID relative to the associated subscription
* `aws_tgw_uid` - The AWS Transit Gateway ID
* `status` - The status of the invitation (e.g., `pending`)
* `aws_account_id` - The AWS account ID associated with the Transit Gateway
