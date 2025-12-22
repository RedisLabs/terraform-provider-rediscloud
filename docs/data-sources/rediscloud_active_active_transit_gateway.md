---
page_title: "Redis Cloud: rediscloud_transit_gateway"
description: |-
  Active Active Transit Gateway data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_transit_gateway

The Active Active Transit Gateway data source allows access to an available Transit Gateway within your Redis Enterprise Cloud Account.

## Example Usage

```hcl
data "rediscloud_active_active_transit_gateway" "example" {
  subscription_id = "113991"
  region_id = 1
  aws_tgw_id = "tgw-1c55bfdoe20pdsad2"
}

output "rediscloud_transit_gateway" {
  value = data.rediscloud_active_active_transit_gateway.example.tgw_id
}
```

## Argument Reference

* `subscription_id` - (Required) The id of an Active Active subscription
* `region_id` - (Required) The id of the AWS region
* `tgw_id` - (Optional) The id of the Transit Gateway relative to the associated subscription. You would likely
  reference this value when creating a `rediscloud_active_active_transit_gateway_attachment`.
* `aws_tgw_id` - (Optional) The id of the Transit Gateway as known to AWS

## Attribute Reference

* `attachment_uid` - A unique identifier for the Subscription/Transit Gateway attachment, if any
* `status` - The status of the Transit Gateway
* `attachment_status` - The status of the Subscription/Transit Gateway attachment, if any
* `aws_account_id` - The Transit Gateway's AWS account id
* `cidrs` - A list of consumer Cidr blocks, if an attachment exists
