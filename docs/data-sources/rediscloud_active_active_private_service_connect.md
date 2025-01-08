---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_private_service_connect"
description: |-
  Active-Active Private Service Connect data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_active_active_private_service_connect

The Active-Active Private Service Connect data source allows access to an available the Private Service Connect Service within your Redis Enterprise Subscription.

## Example Usage

```hcl
data "rediscloud_active_active_private_service_connect" "example" {
  subscription_id = "1234"
  region_id = 1
}

output "rediscloud_psc_status" {
  value = data.rediscloud_active_active_private_service_connect.example.status
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of an Active-Active subscription
* `region_id` - (Required) The ID of the GCP region

## Attribute Reference

* `private_service_connect_service_id` - The ID of the Private Service Connect Service relative to the associated subscription
* `connection_host_name` - The connection hostname
* `service_attachment_name` - The service attachment name
* `status` - The Private Service Connect status
