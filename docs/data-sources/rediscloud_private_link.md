---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_private_link"
description: |-
  PrivateLink data source for Pro Subscription in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_private_link
Retrieves details of an existing PrivateLink for a Pro Subscription.

## Example Usage

```hcl
data "rediscloud_private_link" "example" {
  subscription_id = "1234"
}

output "rediscloud_private_link_principals" {
  value = data.rediscloud_private_link.example.principals
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro Subscription the PrivateLink is attached to.

## Attribute Reference

* `principals` - A list of principals attached to the PrivateLink.
* `resource_configuration_id` - The ID of the resource configuration attached to this PrivateLink.
* `resource_configuration_arn` - The ARN of the resource configuration attached to this PrivateLink.
* `share_arn` - The share ARN of this PrivateLink.
* `connections` - A list of connections associated with the PrivateLink.
* `databases` - A list of databases associated with the PrivateLink.

The `principals` object supports the following attributes:
* `principal` - The principal attached to this PrivateLink.
* `principal_type` - The type of principal.
* `principal_alias` - A friendly name for the principal.

The `connections` object supports the following attributes:
* `association_id` - The association ID of the PrivateLink connection.
* `connection_id` - The connection ID of the PrivateLink connection.
* `connection_type` - The type of the PrivateLink connection.
* `owner_id` - The owner ID of the connection.
* `association_date` - The date the connection was associated.

The `databases` object supports the following attributes:
* `database_id` - The ID of the database.
* `port` - The port the database is available on.
* `resource_link_endpoint` - The resource link endpoint for the database.
