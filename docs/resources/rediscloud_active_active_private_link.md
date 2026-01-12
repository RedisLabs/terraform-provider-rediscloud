---
page_title: "Redis Cloud: rediscloud_active_active_private_link"
description: |-
  PrivateLink resource for an Active-Active Subscription in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_private_link

Manages a PrivateLink to an Active-Active Subscription in your Redis Enterprise Cloud Account.

Note the forced dependency on the Active-Active database. Currently, you require a database to be attached to your subscription in order for a `region_id` to be assigned.

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

resource "rediscloud_active_active_subscription" "aa_subscription" {
  name              = local.rediscloud_subscription_name
  payment_method    = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "eu-west-1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "eu-west-2"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "aa_database" {
  subscription_id         = rediscloud_active_active_subscription.aa_subscription.id
  name                    = "db"
  memory_limit_in_gb      = 1
  global_data_persistence = "aof-every-1-second"
  global_password         = local.rediscloud_database_password
}

data "rediscloud_active_active_subscription_regions" "aa_regions_info" {
  subscription_name = rediscloud_active_active_subscription.aa_subscription.name
  depends_on        = [rediscloud_active_active_subscription_database.aa_database]
}


resource "rediscloud_active_active_private_link" "private_link" {
  subscription_id = rediscloud_active_active_subscription.aa_subscription.id
  region_id       = data.rediscloud_active_active_subscription_regions.aa_regions_info.regions[0].region_id
  share_name      = local.rediscloud_private_link_share_name

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
```

## Argument Reference

* `subscription_id` - (Required) The ID of the Active-Active Subscription to link to.  **Modifying this attribute will force creation of a new resource.**
* `region_id` - (Required) The region ID within the Active-Active subscription that the PrivateLink is attached to. **Modifying this attribute will force creation of a new resource.**
* `share_name` - (Required) The share name of the PrivateLink.
* `principal` - (Required) The principal(s) attached to the PrivateLink.

The `principal` block supports:
* `principal` - (Required) The principal to be added to this PrivateLink. The format depends upon the type of principal you wish to attach.
* `principal_type` - (Required) The principal type. Can be one of: `aws_account`, `organization`, `organization_unit`, `iam_role`, `iam_user`, `service_principal`.
* `principal_alias` - The friendly name to refer to the principal.

## Attribute Reference

* `resource_configuration_id` - ID of the resource configuration to attach to this PrivateLink
* `resource_configuration_arn` - ARN of the resource configuration to attach to this PrivateLink
* `share_arn` - Share ARN of this PrivateLink.
* `connections` - List of connections associated with the PrivateLink.
* `databases` - List of databases associated with the PrivateLink.

The `connections` object has these attributes:

* `association_id` - Association ID of the PrivateLink connection.
* `connection_id` - Connection ID of the PrivateLink connection.
* `connection_type` - Type of the PrivateLink connection.
* `owner_id` - Owner ID of the connection.
* `association_date` - Date the connection was associated.

The `databases` object has these attributes:
* `database_id` - ID of the database.
* `port` - The port which the database is available on.
* `resource_link_endpoint` - The resource link endpoint for the database.

## Import
`rediscloud_active_active_private_link` can be imported using the ID of the subscription and the region id in the format SUB_ID/REGION_ID, e.g.

```
$ terraform import rediscloud_active_active_private_link.id 123456/1
```
