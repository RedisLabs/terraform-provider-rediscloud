---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_transit_gateway_attachment"
description: |-
  Transit Gateway Attachment resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_transit_gateway_attachment

Manages a Transit Gateway Attachment to a Pro/Flexible Subscription in your Redis Enterprise Cloud Account.

```hcl
data "rediscloud_transit_gateway" "gateway" {
  subscription_id = "113491"
  aws_tgw_id = "tgw-1c55bfdoe20pdsad2"
}

resource "rediscloud_transit_gateway_attachment" "attachment" {
  subscription_id = "113491"
  tgw_id = data.rediscloud_transit_gateway.gateway.tgw_id
}
```

## Argument Reference

* `subscription_id` - (Required) The id of the Pro/Flexible subscription to attach
* `tgw_id` - (Required) The id of the Transit Gateway to attach to
* `cidrs` - (Optional) A list of consumer Cidr blocks. **May only be added once the Attachment has been accepted**

## Attribute Reference

* `aws_tgw_uid` - The id of the Transit Gateway as known to AWS
* `attachment_uid` - A unique identifier for the Subscription/Transit Gateway attachment, if established
* `status` - The status of the Transit Gateway
* `attachment_status` - The status of the Subscription/Transit Gateway attachment, if established
* `aws_account_id` - The Transit Gateway's AWS account id

## Import
`rediscloud_transit_gateway_attachment` can be imported using the ID of the Pro/Flexible subscription and the ID of the Transit Gateway in the format {subscription ID}/{tgw ID}, e.g.

```
$ terraform import rediscloud_transit_gateway_attachment.tgwa-resource 123456/47
```
