---
page_title: "Redis Cloud: rediscloud_active_active_private_link_endpoint_script"
description: |-
  PrivateLink endpoint script data source for Active-Active subscriptions in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_active_active_private_link_endpoint_script
Retrieves the PrivateLink endpoint script for an Active-Active subscription region. This script can be used to configure your VPC endpoint to connect to the Redis Cloud PrivateLink.

## Example Usage

```hcl
data "rediscloud_active_active_private_link_endpoint_script" "example" {
  subscription_id = "1234"
  region_id       = 1
}

output "rediscloud_private_link_endpoint_script" {
  value = data.rediscloud_active_active_private_link_endpoint_script.example.endpoint_script
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Active-Active subscription the PrivateLink is attached to.
* `region_id` - (Required) The region ID within the Active-Active subscription that the PrivateLink is attached to.

## Attribute Reference

* `endpoint_script` - The endpoint script for configuring the PrivateLink connection.
