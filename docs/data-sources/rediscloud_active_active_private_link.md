---
page_title: "Redis Cloud: rediscloud_active_active_private_link"
description: |-
  PrivateLink data source for Active-Active subscriptions in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_active_active_private_link
Retrieves information about an existing PrivateLink for an Active-Active subscription region.

## Example Usage

```hcl
data "rediscloud_active_active_private_link" "example" {
  subscription_id = "1234"
  region_id       = 1
}

output "rediscloud_private_link_principals" {
  value = data.rediscloud_active_active_private_link.example.principals
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Active-Active subscription the PrivateLink is attached to.
* `region_id` - (Required) The region ID within the Active-Active subscription that the PrivateLink is attached to.

## Attribute Reference

* `principals` - A list of principals attached to the PrivateLink.
* `resource_configuration_id` - The ID of the resource configuration attached to this PrivateLink.
* `resource_configuration_arn` - The ARN of the resource configuration attached to this PrivateLink.
* `share_arn` - The share ARN of this PrivateLink.
* `connections` - A list of connections associated with the PrivateLink.
* `databases` - A list of databases associated with the PrivateLink.

The `principals` object is a list with these attributes:
* `principal` - The principal attached to this PrivateLink.
* `principal_type` - The type of principal.
* `principal_alias` - A friendly name for the principal.

The `connections` object is a list with these attributes:
* `association_id` - The association ID of the PrivateLink connection.
* `connection_id` - The connection ID of the PrivateLink connection.
* `connection_type` - The type of the PrivateLink connection.
* `owner_id` - The owner ID of the connection.
* `association_date` - The date the connection was associated.

The `databases` object is a list with these attributes:
* `database_id` - The ID of the database.
* `port` - The port the database is available on.
* `resource_link_endpoint` - The resource link endpoint for the database.
