---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_transit_gateway_invitations"
description: |-
  Active-Active Transit Gateway Invitations data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_active_active_transit_gateway_invitations

The Active-Active Transit Gateway Invitations data source allows access to pending Transit Gateway invitations within an Active-Active Redis Enterprise Cloud Subscription region. These invitations are created when an AWS Resource Share shares a Transit Gateway with your subscription.

## Example Usage

```hcl
data "rediscloud_active_active_transit_gateway_invitations" "example" {
  subscription_id = "113991"
  region_id       = 1
}

output "pending_invitations" {
  value = data.rediscloud_active_active_transit_gateway_invitations.example.invitations
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of an Active-Active subscription
* `region_id` - (Required) The ID of the region within the Active-Active subscription

## Attribute Reference

* `invitations` - List of Transit Gateway invitations, documented below

The `invitations` object has these attributes:

* `id` - The ID of the Transit Gateway invitation
* `name` - The name of the resource share
* `resource_share_uid` - The AWS Resource Share ARN
* `aws_account_id` - The AWS account ID that shared the Transit Gateway
* `status` - The status of the invitation (e.g., `pending`, `accepted`, `rejected`)
* `shared_date` - The date the resource was shared
