---
page_title: "Redis Cloud: rediscloud_active_active_private_service_connect"
description: |-
  Private Service Connect resource for Active-Active Subscription in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_private_service_connect

Manages a Private Service Connect to an Active-Active Subscription in your Redis Enterprise Cloud Account.

## Example Usage

[Full example in the `rediscloud_active_active_private_service_connect_endpoint` resource](./rediscloud_active_active_private_service_connect_endpoint.md)

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro subscription to attach **Modifying this attribute will force creation of a new resource.**
* `region_id` - (Required) The ID of the region, as created by the API **Modifying this attribute will force creation of a new resource.**

## Attribute Reference

* `private_service_connect_service_id` - The ID of the Private Service Connect Service relative to the associated subscription

## Import

`rediscloud_active_active_private_service_connect` can be imported using the ID of the Active-Active subscription, the region ID and the ID of the Private Service Connect in the format {subscription ID/region ID/private service connect ID}, e.g.

```
$ terraform import rediscloud_active_active_private_service_connect.id 1000/1/123456
```
