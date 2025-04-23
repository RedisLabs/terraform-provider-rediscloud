---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_private_service_connect_endpoint_accepter"
description: |-
  Private Service Connect Endpoint Accepter resource for Active-Active Subscription in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_private_service_connect_endpoint_accepter

Manages a Private Service Connect Endpoint state in your Redis Enterprise Cloud Account.

## Example Usage

[Full example in the `rediscloud_active_active_private_service_connect_endpoint` resource](./rediscloud_active_active_private_service_connect_endpoint.md)

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro subscription to attach **Modifying this attribute will force creation of a new resource.**
* `region_id` - (Required) The ID of the region, as created by the API **Modifying this attribute will force creation of a new resource.**
* `private_service_connect_service_id` - (Required) The ID of the Private Service Connect Service relative to the associated subscription **Modifying this attribute will force creation of a new resource.**
* `private_service_connect_endpoint_id` - (Required) The ID of the Private Service Connect Service relative to the associated subscription **Modifying this attribute will force creation of a new resource.**
* `action` - (Required) Accept or reject the endpoint (accepted values are `accept` and `reject`)
