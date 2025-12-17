---
page_title: "Redis Cloud: rediscloud_private_link_endpoint_script"
description: |-
  PrivateLink endpoint script data source for Pro subscriptions in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_private_link_endpoint_script
Retrieves the PrivateLink endpoint script for a Pro subscription. This script can be used to configure your VPC endpoint to connect to the Redis Cloud PrivateLink.

## Example Usage

```hcl
data "rediscloud_private_link_endpoint_script" "example" {
  subscription_id = "1234"
}

output "rediscloud_private_link_endpoint_script" {
  value = data.rediscloud_private_link_endpoint_script.example.endpoint_script
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro subscription the PrivateLink is attached to.

## Attribute Reference

* `endpoint_script` - The endpoint script for configuring the PrivateLink connection.
