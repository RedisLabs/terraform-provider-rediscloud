---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_subscription_database"
description: |-
Database resource for Active-Active Subscripitons in the Terraform provider Redis Cloud.
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

resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "example-database"
    memory_limit_in_gb = 1
    support_oss_cluster_api = false 
    external_endpoint_for_oss_cluster_api = false
	enable_tls = false
	data_eviction = "volatile-lru"
    
    // OPTIONAL
    global_data_persistence = "aof-every-1-second"
    global_password = "some-random-pass-2" 
    global_alert {
		name = "dataset-size"
		value = 40
	}
	// TODO: add source_ips example
	

  override_region {
    name = "us-east-2"
  }

  override_region {
    name = "us-east-1"
    override_global_data_persistence = "none"
	// TODO: add global_source_ips example
    # override_global_source_ips = []
    override_global_password = "region-specific-password"
    override_global_alert {
        name = "dataset-size"
        value = 41
    }
   }
}

output "us-east-public" {
  value = rediscloud_active_active_subscription_database.example.public_endpoint.us-east-1
}

output "all-private-endpoints" {
  value = rediscloud_active_active_subscription_database.example.private_endpoint
}
```

## Argument Reference

The following arguments are supported:
* `subscription_id`: (Required) The ID of the subscription to create the database in.
* `name` - (Required) A meaningful name to identify the database.
  the top of the page for more information.
* `memory_limit_in_gb` - (Required) Maximum memory usage for this specific database
* `support_oss_cluster_api` - (Optional) Support Redis open-source (OSS) Cluster API. Default: ‘false’
* `external_endpoint_for_oss_cluster_api` - (Optional) Should use the external endpoint for open-source (OSS) Cluster API.
  Can only be enabled if OSS Cluster API support is enabled. Default: 'false'
* `enable_tls` - (Optional) Use TLS for authentication. Default: ‘false’
* `client_ssl_certificate` - (Optional) SSL certificate to authenticate user connections.
* `data_eviction` - (Optional) The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru')
* `global_data_persistence` - (Optional) Rate of database data persistence (in persistent storage).
* `global_password` - (Optional) Password used to access the database. If left empty, the password will be generated automatically.
* `global_alert` - (Optional) A block defining an alert to enable on the database, documented below, can be specified multiple times.
* `global_source_ips` - (Optional) List of CIDR addresses to allow access to the database.
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
* `public_endpoint` - A map of which public endpoints can to access the database per region, uses region name as key.
* `private_endpoint` - A map of which private endpoints can to access the database per region, uses region name as key.

## Import
`rediscloud_active_active_subscription_database` can be imported using the ID of the subscription and the ID of the database in the format {subscription ID}/{database ID}, e.g.

```
$ terraform import rediscloud_active_active_subscription_database.example_database 123456/12345678
```

