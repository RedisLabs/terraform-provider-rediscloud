---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_private_link"
description: |-
  PrivateLink resource for Active Active Subscription in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_activeprivate_link

Manages a PrivateLink to a Active Active Subscription in your Redis Enterprise Cloud Account.

## Example Usage

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro Subscription to  **Modifying this attribute will force creation of a new resource.**
* `share_name` - (Required)
* `principal` - (Required) (block)
* `region_id` - (Required)

## Attribute Reference

* `resource_configuration_id`
* `resource_configuration_arn`
* `share_arn`
* `connections` (block)
* `databases` (block)

## Import
`rediscloud_private_link` can be imported using the ID of the subscription and the name of the PrivateLink share, e.g.

```
$ terraform import rediscloud_active_active_private_link.id 1000/123456
```
