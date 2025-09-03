---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_private_link"
description: |-
  PrivateLink resource for Pro Subscription in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_private_link

Manages a PrivateLink to a Pro Subscription in your Redis Enterprise Cloud Account.

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
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro Subscription to attach the PrivateLink to.  **Modifying this attribute will force creation of a new resource.**
* `share_name` - (Required) The share name of the PrivateLink.
* `principal` - (Required) The principal(s) attached to the PrivateLink.

The `principal` block supports:
* `principal` - (Required) The principal to be added to this PrivateLink. The format depends upon the type of principal you wish to attach.
* `principal_type` - (Required) The principal type. Can be one of: `aws_account`, `organization`, `organization_unit`, `iam_role`, `iam_user`, `service_principal`.
* `principal_alias` - The friendly name to refer to the principal.


## Attribute Reference

* `resource_configuration_id`
* `resource_configuration_arn`
* `share_arn`
* `connections`
* `databases`

The `connections` object has these attributes:

* `association_id` - Association ID of the PrivateLink connection.
* `connection_id` - Connection ID of the PrivateLink connection
* `connection_type` - Type of the PrivateLink connection.
* `owner_id` - Owner ID of the connection.
* `association_date` - Date the connection was associated.

The `databases` object has these attributes:
* `database_id` - ID of the database.
* `port` - The port which the database is available on.
* `resource_link_endpoint` - The resource link endpoint for the database.

## Import
`rediscloud_private_link` can be imported using the ID of the subscription and the ID of the PrivateLink, e.g.

```
$ terraform import rediscloud_private_service_connect.id 1000/123456
```
