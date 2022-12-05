---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_subscription_database"
description: |-
Database resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_active_active_subscription_database

Creates a Database within a specified Active Active Subscription in your Redis Enterprise Cloud Account.

## Example Usage

```hcl
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "example" {
  name = "%s" 
  payment_method_id = data.rediscloud_payment_method.card.id 
  cloud_provider = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=true
	region {
		region = "us-east-1"
		networking_deployment_cidr = "192.168.0.0/24"
		write_operations_per_second = 1000
		read_operations_per_second = 1000
	}
	region {
		region = "us-east-2"
		networking_deployment_cidr = "10.0.1.0/24"
		write_operations_per_second = 1000
		read_operations_per_second = 1000
	}
	}
  }
}

resource "rediscloud_subscription_active_active_database" "example" {
    subscription_id              = rediscloud_subscription.example.id
    name                         = "tf-database"
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    global_data_persistence      = "none"
	global_password              = "%s"
	support_oss_cluster_api	     = true
}

data "rediscloud_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = rediscloud_subscription_database.example.name
}
```

## Argument Reference

The following arguments are supported:

* `subscription_id`: (Required) The ID of the subscription to create the database in.
* `name` - (Required) A meaningful name to identify the database.
  the top of the page for more information.
* `memory_limit_in_gb` - (Required) Maximum memory usage for this specific database
* `protocol` - (Optional) The protocol that will be used to access the database, (either ‘redis’ or 'memcached’) Default: ‘redis’
* `support_oss_cluster_api` - (Optional) Support Redis open-source (OSS) Cluster API. Default: ‘false’
* `external_endpoint_for_oss_cluster_api` - (Optional) Should use the external endpoint for open-source (OSS) Cluster API.
  Can only be enabled if OSS Cluster API support is enabled. Default: 'false'
* `global_data_persistence` - (Optional) Rate of database data persistence (in persistent storage).
* `global_password` - (Optional) Password used to access the database. If left empty, the password will be generated automatically.
* `replica_of` - (Optional) Set of Redis database URIs, in the format `redis://user:password@host:port`, that this
  database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided.
  Cannot be enabled when `support_oss_cluster_api` is enabled.
* `global_source_ips` - (Optional) Set of CIDR addresses to allow access to the database.
* `override_global_alert` - (Optional) Set of alerts to enable on the database, documented below.
* `override_region` - (Optional) Override region specific configuration, documented below.

The `override_global_alert` block supports:

* `name` - (Required) Alert name
* `value` - (Required) Alert value

The `override_region` block supports:

* `name` - (Required) Region name.
* `override_global_alert` - (Optional) Set of alerts to enable on the database.
* `override_global_password` - (Optional) Password used to access the database. If left empty, the password will be generated automatically.
* `override_global_source_ips` - (Optional) Set of CIDR addresses to allow access to the database.
* `override_global_data_persistence` - (Optional) Rate of database data persistence (in persistent storage).

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 30 mins) Used when creating the database
* `update` - (Defaults to 30 mins) Used when updating the database
* `delete` - (Defaults to 10 mins) Used when destroying the database

## Attribute reference

* `db_id` - Identifier of the database created
* `public_endpoint` - Public endpoint to access the database
* `private_endpoint` - Private endpoint to access the database

## Import
`rediscloud_subscription_database` can be imported using the ID of the subscription and the ID of the database in the format {subscription ID}/{database ID}, e.g.

```
$ terraform import rediscloud_subscription_database.example_database 123456/12345678
```

