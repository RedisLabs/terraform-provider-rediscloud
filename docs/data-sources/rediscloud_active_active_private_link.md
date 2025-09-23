---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_private_link"
description: |-
  PrivateLink data source for Pro Subscription in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_active_active_private_link
The PrivateLink data source allows the user to retrieve information about an existing PrivateLink for an Active Active Subscription in the provider.

## Example Usage

```hcl

locals {
  rediscloud_subscription_name = "..."
  rediscloud_cloud_account = "..."
  rediscloud_private_link_share_name = "..."
  rediscloud_database_password = "..."
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = local.rediscloud_cloud_account
}

resource "rediscloud_active_active_subscription" "subscription" {
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

resource "rediscloud_active_active_subscription_database" "database_resource" {
  subscription_id         = rediscloud_active_active_subscription.subscription.id
  name                    = "db"
  memory_limit_in_gb      = 1
  global_data_persistence = "aof-every-1-second"
  global_password         = local.rediscloud_database_password
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
    principal = "234567890123"
    principal_type = "aws_account"
    principal_alias = "principal 2"
  }
}

data "rediscloud_active_active_private_link" "private_link" {
  subscription_id = rediscloud_active_active_private_link.private_link_1.subscription_id
  region_id = rediscloud_active_active_private_link.private_link_1.region_id
}

```

## Argument Reference

* `subscription_id` - (Required) The ID of the Active Active Subscription the PrivateLink is attached to.
* `region_id` - (Required) The region ID within the Active Active subscription that the PrivateLink is attached to.

## Attribute reference

* `principals` - The principal(s) attached to the PrivateLink.
* `resource_configuration_id` - ID of the resource configuration to attach to this PrivateLink
* `resource_configuration_arn` - ARN of the resource configuration to attach to this PrivateLink
* `share_arn` - Share ARN of this PrivateLink.
* `connections` - List of connections associated with the PrivateLink.
* `databases` - List of databases associated with the PrivateLink.

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
