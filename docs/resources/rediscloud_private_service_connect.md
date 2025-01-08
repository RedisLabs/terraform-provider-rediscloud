---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_private_service_connect"
description: |-
  Private Service Connect resource for Pro Subscription in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_private_service_connect

Manages a Private Service Connect to a Pro Subscription in your Redis Enterprise Cloud Account.

## Example Usage

[Full example in the `rediscloud_private_service_connect_endpoint` resource](./rediscloud_private_service_connect_endpoint.md)

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro subscription to attach **Modifying this attribute will force creation of a new resource.**

## Attribute Reference

* `private_service_connect_service_id` - The ID of the Private Service Connect Service relative to the associated subscription

## Import
`rediscloud_private_service_connect` can be imported using the ID of the subscription and the ID of the Private Service Connect in the format {subscription ID/private service connect ID}, e.g.

```
$ terraform import rediscloud_private_service_connect.id 1000/123456
```
