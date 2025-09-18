---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_private_link"
description: |-
  PrivateLink data source for Pro Subscription in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_private_link
The PrivateLink data source allows the user to retrieve information about an existing PrivateLink in the provider.

## Example Usage

```hcl

locals {
  rediscloud_subscription_name = "..."
  rediscloud_cloud_account = "..."
  rediscloud_private_link_share_name = "..."
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = local.rediscloud_cloud_account
}

resource "rediscloud_subscription" "subscription" {
  name              = local.rediscloud_subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    dataset_size_in_gb           = 15
    quantity                     = 1
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 20000
  }
}

resource "rediscloud_private_link" "private_link" {
  subscription_id = rediscloud_subscription.subscription.id
  share_name = local.rediscloud_private_link_share_name

  principal {
    principal = "123456789012"
    principal_type = "aws_account"
    principal_alias = "principal 1"
  }

  principal {
    principal = "123456789013"
    principal_type = "aws_account"
    principal_alias = "principal 2"
  }
}

data "rediscloud_private_link" "private_link" {
  subscription_id = rediscloud_private_link.private_link.subscription_id
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro Subscription the PrivateLink is attached to.

## Attribute reference

* `principals` - The principal(s) attached to the PrivateLink.
* `resource_configuration_id`
* `resource_configuration_arn`
* `share_arn`
* `connections`
* `databases`

The `principals` object is a list, with these attributes:
* `principal` - The principal attached to this PrivateLink.
* `principal_type` - The principal type.
* `principal_alias` - The friendly name to refer to the principal.

The `connections` object is a list, with these attributes:
* `association_id` - Association ID of the PrivateLink connection.
* `connection_id` - Connection ID of the PrivateLink connection
* `connection_type` - The PrivateLink connection type.
* `owner_id` - Owner ID of the connection.
* `association_date` - Date the connection was associated.

The `databases`  object is a list, with these attributes:
* `database_id` - ID of the database.
* `port` - The port which the database is available on.
* `resource_link_endpoint` - The resource link endpoint for the database.
