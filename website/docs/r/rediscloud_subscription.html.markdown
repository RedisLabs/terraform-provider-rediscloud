---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription"
description: |-
  Subscription resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_subscription

Creates a Subscription within your Redis Enterprise Cloud Account.
This resource is responsible for creating subscriptions and the databases within that subscription. 
This allows your Redis Enterprise Cloud Account to efficiently pack the databases within provisioned cloud provider compute instances.

~> **Note:** The subscription resource manages changes to its databases by identifying a databases through its name.  This means **database names cannot be changed**, as this resource has no other way of being able to identify the database and would lead to the database to be destroyed.
Due to the limitations mentioned above, the differences shown by Terraform will be different from normal plan.
When an argument has been changed on a nested database - for example changing the `memory_limit_in_gb` from 1 to 2, Terraform
will display the resource as being modified with the database as being removed, and a new one added. As the resource
identifies the database based on the name, the only change that would happen would be to update the database to increase
the memory limit. Below is the Terraform output for changing the `memory_limit_in_gb` for a single database within a
subscription.

```
An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  ~ update in-place

Terraform will perform the following actions:

  # rediscloud_subscription.example will be updated in-place
  ~ resource "rediscloud_subscription" "example" {

        ...

      - database {
          - average_item_size_in_bytes            = 0 -> null
          - data_persistence                      = "none" -> null
          - db_id                                 = 51040112 -> null
          - external_endpoint_for_oss_cluster_api = false -> null
          - memory_limit_in_gb                    = 1 -> null
          - name                                  = "tf-example-database" -> null
          - password                              = (sensitive value)
          - private_endpoint                      = "private.example.com" -> null
          - protocol                              = "redis" -> null
          - public_endpoint                       = "public.example.com" -> null
          - replica_of                            = [] -> null
          - replication                           = false -> null
          - source_ips                            = [] -> null
          - support_oss_cluster_api               = false -> null
          - throughput_measurement_by             = "operations-per-second" -> null
          - throughput_measurement_value          = 10000 -> null
        }
      + database {
          + average_item_size_in_bytes            = 0
          + data_persistence                      = "none"
          + db_id                                 = (known after apply)
          + external_endpoint_for_oss_cluster_api = false
          + memory_limit_in_gb                    = 2
          + name                                  = "tf-example-database"
          + password                              = (sensitive value)
          + private_endpoint                      = (known after apply)
          + protocol                              = "redis"
          + public_endpoint                       = (known after apply)
          + replica_of                            = []
          + replication                           = false
          + source_ips                            = []
          + support_oss_cluster_api               = false
          + throughput_measurement_by             = "operations-per-second"
          + throughput_measurement_value          = 10000
        }
    }

Plan: 0 to add, 1 to change, 0 to destroy.
```

## Example Usage

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
}

resource "random_password" "password" {
  length = 20
  upper = true
  lower = true
  number = true
  special = false
}

resource "rediscloud_subscription" "example" {

  name = "example"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  persistent_storage_encryption = false

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  database {
    name = "tf-example-database"
    protocol = "redis"
    memory_limit_in_gb = 1
    data_persistence = "none"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    password = random_password.password.result

    alert {
      name = "dataset-size"
      value = 40
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A meaningful name to identify the subscription
* `payment_method_id` - (Required) A valid payment method pre-defined in the current account
* `memory_storage` - (Optional) Memory storage preference: either ‘ram’ or a combination of 'ram-and-flash’. Default: ‘ram’
* `persistent_storage_encryption` - (Optional) Encrypt data stored in persistent storage. Required for a GCP subscription. Default: ‘false’
* `allowlist` - (Optional) An allowlist object, documented below 
* `cloud_provider` - (Required) A cloud provider object, documented below 
* `database` - (Required) A database object, documented below

The `allowlist` block supports:

* `cidrs` - (Optional) Set of CIDR ranges that are allowed to access the databases associated with this subscription
* `security_group_ids` - (Optional) Set of security groups that are allowed to access the databases associated with this subscription

The `cloud_provider` block supports:

* `provider` - (Optional) The cloud provider to use with the subscription, (either `AWS` or `GCP`). Default: ‘AWS’
* `cloud_account_id` - (Optional) Cloud account identifier. Default: Redis Labs internal cloud account
(using Cloud Account Id = 1 implies using Redis Labs internal cloud account). Note that a GCP subscription can be created
only with Redis Labs internal cloud account.
* `region` - (Required) Cloud networking details, per region (single region or multiple regions for Active-Active cluster only),
documented below

The `database` block supports:

* `name` - (Required) A meaningful name to identify the database. Caution should be taken when changing this value - see
the top of the page for more information.
* `protocol` - (Optional) The protocol that will be used to access the database, (either ‘redis’ or 'memcached’) Default: ‘redis’
* `memory_limit_in_gb` - (Required) Maximum memory usage for this specific database
* `support_oss_cluster_api` - (Optional) Support Redis open-source (OSS) Cluster API. Default: ‘false’
* `external_endpoint_for_oss_cluster_api` - (Optional) Should use the external endpoint for open-source (OSS) Cluster API.
Can only be enabled if OSS Cluster API support is enabled. Default: 'false'
* `client_ssl_certificate` - (Optional) SSL certificate to authenticate user connections
* `periodic_backup_path` - (Optional) Path that will be used to store database backup files
* `replica_of` - (Optional) Set of Redis database URIs, in the format `redis://user:password@host:port`, that this
database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided.
Cannot be enabled when `support_oss_cluster_api` is enabled.
* `module` - (Optional) A module object, documented below
* `alert` - (Optional) Set of alerts to enable on the database, documented below
* `data_persistence` - (Optional) Rate of database data persistence (in persistent storage). Default: ‘none’
* `password` - (Required) Password used to access the database
* `replication` - (Optional) Databases replication. Default: ‘true’
* `throughput_measurement_by` - (Required) Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)
* `throughput_measurement_value` - (Required) Throughput value (as applies to selected measurement method)
* `average_item_size_in_bytes` - (Optional) Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes)
of the items stored in the database. Default: 1000
* `source_ips` - (Optional) Set of CIDR addresses to allow access to the database. Defaults to allowing traffic.
* `hashing_policy` - (Optional) List of regular expression rules to shard the database by. See
[the documentation on clustering](https://docs.redislabs.com/latest/rc/concepts/clustering/) for more information on the
hashing policy. This cannot be set when `support_oss_cluster_api` is set to true.

The cloud_provider `region` block supports:

* `region` - (Required) Deployment region as defined by cloud provider
* `multiple_availability_zones` - (Optional) Support deployment on multiple availability zones within the selected region. Default: ‘false’
* `networking_deployment_cidr` - (Required) Deployment CIDR mask. Default: If using Redis Labs internal cloud account, 192.168.0.0/24
* `networking_vpc_id` - (Optional) Either an existing VPC Id (already exists in the specific region) or create a new VPC
(if no VPC is specified). VPC Identifier must be in a valid format (for example: ‘vpc-0125be68a4625884ad’) and existing
within the hosting account.
* `preferred_availability_zones` - (Required) Availability zones deployment preferences (for the selected provider & region).

The database `alert` block supports:

* `name` (Required) Alert name
* `value` (Required) Alert value

The database `module` block supports:

* `name` (Required) Name of the module to enable

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 mins) Used when creating the subscription
* `update` - (Defaults to 10 mins) Used when updating the subscrition
* `delete` - (Defaults to 10 mins) Used when destroying the subscription

## Attribute reference

The `database` block has these attributes:

* `db_id` - Identifier of the database created
* `public_endpoint` - Public endpoint to access the database
* `private_endpoint` - Private endpoint to access the database

The `region` block has these attributes:

* `networking_subnet_id` - The subnet that the subscription deploys into

## Import

`rediscloud_subscription` can be imported using the ID of the subscription, e.g.

```
$ terraform import rediscloud_subscription.example 12345678
```
