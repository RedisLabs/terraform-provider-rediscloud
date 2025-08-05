---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription"
description: |-
  Subscription resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_subscription

This resource allows you to manage a subscription within your Redis Enterprise Cloud account.

-> **Note:** This is for Pro Subscriptions only. See also `rediscloud_active_active_subscription` and `rediscloud_essentials_subscription`.

~> **Note:** The payment_method property is ignored after Subscription creation.

~> **Note:** The creation_plan block allows the API server to create a well-optimised infrastructure for your databases in the cluster.
The attributes inside the block are used by the provider to create initial
databases. Those databases will be deleted after provisioning a new
subscription, then the databases defined as separate resources will be attached to
the subscription. The creation_plan block can ONLY be used for provisioning new
subscriptions, the block will be ignored if you make any further changes or try importing the resource (e.g. `terraform import` ...).

~> **Note:** The CMK (customer managed encryption key) fields require a specific flow which involves a multistep apply. Refer to [this guide](../guides/cmk-guide.md) for more information.
## Example Usage

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "subscription-resource" {

  name              = "subscription-name"
  payment_method    = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"
  redis_version     = "7.2"

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    region {
      region                       = "eu-west-1"
      multiple_availability_zones  = true
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = ["euw1-az1", "euw1-az2", "euw1-az3"]
    }
  }

  // This block needs to be defined for provisioning a new subscription.
  // This allows creation of a well-optimized hardware specification for databases in the cluster
  creation_plan {
    dataset_size_in_gb           = 15
    quantity                     = 1
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 20000
    modules                      = ["RedisJSON"]
  }
  
  maintenance_windows {
    mode = "manual"
    window {
      start_hour = 22
      duration_in_hours = 8
      days = ["Tuesday", "Friday"]
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A meaningful name to identify the subscription
* `payment_method` (Optional) The payment method for the requested subscription, (either `credit-card` or `marketplace`). If `credit-card` is specified, `payment_method_id` must be defined. Default: 'credit-card'. **(Changes to) this attribute are ignored after creation.**
* `payment_method_id` - (Optional) A valid payment method pre-defined in the current account. Only __Required__ when `payment_method` is `credit-card`.
* `memory_storage` - (Optional) Memory storage preference: either ‘ram’ or a combination of ‘ram-and-flash’. Default: ‘ram’. **Modifying this attribute will force creation of a new resource.**
* `redis_version` - (Optional) The Redis version of the databases in the subscription. If omitted, the Redis version will be the default. **Modifying this attribute will force creation of a new resource.**
* `allowlist` - (Optional) An allowlist object, documented below
* `cloud_provider` - (Required) A cloud provider object, documented below. **Modifying this attribute will force creation of a new resource.**
* `creation_plan` - (Required) A creation plan object, documented below.
* `maintenance_windows` - (Optional) The subscription's maintenance window specification, documented below.
* `customer_managed_key_enabled` - (Optional) Whether to enable the customer managed encryption key flow.
* `customer_managed_key_deletion_grace_period` - (Optional) The grace period for deleting the subscription. If not set, will default to immediate deletion grace period.
* `customer_managed_key` - (Optional) The customer managed keys (CMK) to use for this subscription. If is active-active subscription, must set a key for each region.

The `allowlist` block supports:

* `security_group_ids` - (Required) Set of security groups that are allowed to access the databases associated with this subscription
* `cidrs` - (Optional) Set of CIDR ranges that are allowed to access the databases associated with this subscription

~> **Note:** `allowlist` is only available when you run on your own cloud account, and not one that Redis provided (i.e `cloud_account_id` != 1)

The `cloud_provider` block supports:

* `provider` - (Optional) The cloud provider to use with the subscription, (either `AWS` or `GCP`). Default: ‘AWS’. **Modifying this attribute will force creation of a new resource.**
* `cloud_account_id` - (Optional) Cloud account identifier. Default: Redis Labs internal cloud account. **Modifying this attribute will force creation of a new resource.**
  (using Cloud Account ID = 1 implies using Redis Labs internal cloud account). Note that a GCP subscription can be created
  only with Redis Labs internal cloud account
* `region` - (Required) A region object, documented below. **Modifying this attribute will force creation of a new resource.**

The `creation_plan` block supports:

* `memory_limit_in_gb` - (Required) Maximum memory usage that will be used for your largest planned database. You can not set both dataset_size_in_gb and memory_limit_in_gb. **Deprecated: Use `dataset_size_in_gb` instead**
* `dataset_size_in_gb` - (Required) The maximum amount of data in the dataset for this specific database is in GB. You can not set both dataset_size_in_gb and memory_limit_in_gb.
* `modules` - (Optional) a list of modules that will be used by the databases in this subscription. Not currently compatible with ‘ram-and-flash’ memory storage.  
  Example: `modules = ["RedisJSON", "RediSearch", "RedisTimeSeries", "RedisBloom"]`
* `support_oss_cluster_api` - (Optional) Support Redis open-source (OSS) Cluster API. Default: ‘false’
* `replication` - (Optional) Databases replication. Default: ‘true’
* `quantity` - (Required) The planned number of databases in the subscription
* `throughput_measurement_by` - (Required) Throughput measurement method that will be used by your databases. Either `number-of-shards` or `operations-per-second`. **`number-of-shards` is deprecated and only supported for legacy deployments.**
* `throughput_measurement_value` - (Required) Throughput value that will be used by your databases (as applies to selected measurement method). The value needs to be the maximum throughput measurement value defined in one of your databases
* `average_item_size_in_bytes` - (Optional) Relevant only to ram-and-flash clusters
  Estimated average size (measured in bytes) of the items stored in the database. The value needs to
  be the maximum average item size defined in one of your databases.  Default: 1000

~> **Note:** If the number of modules exceeds the `quantity` then additional creation-plan databases will be created with the modules defined in the `modules` block.

~> **Note:** If changes are made to attributes in the subscription which require the subscription to be recreated (such as `memory_storage` or `cloud_provider`), the creation_plan will need to be defined in order to change these attributes. This is because the creation_plan is always required when a subscription is created.

The cloud_provider `region` block supports:

* `region` - (Required) Deployment region as defined by cloud provider. **Modifying this attribute will force creation of a new resource.**
* `multiple_availability_zones` - (Optional) Support deployment on multiple availability zones within the selected region. Default: ‘false’. **Modifying this attribute will force creation of a new resource.**
* `networking_deployment_cidr` - (Required) Deployment CIDR mask. The total number of bits must be 24 (x.x.x.x/24). **Modifying this attribute will force creation of a new resource.**
* `networking_vpc_id` - (Optional) Either an existing VPC Id (already exists in the specific region) or create a new VPC
  (if no VPC is specified). VPC Identifier must be in a valid format (for example: ‘vpc-0125be68a4986384ad’) and exist
  within the hosting account. **Modifying this attribute will force creation of a new resource.**
* `preferred_availability_zones` - (Optional) Availability zones deployment preferences (for the selected provider & region). If multiple_availability_zones is set to 'true', select three availability zones from the list. If you don't want to specify preferred availability zones, set this attribute to an empty list ('[]'). **Modifying this attribute will force creation of a new resource.**

~> **Note:** The preferred_availability_zones parameter is required for Terraform, but is optional within the Redis Enterprise Cloud UI.
This difference in behaviour is to guarantee that a plan after an apply does not generate differences. In AWS Redis internal cloud account, please set the zone IDs (for example: `["use-az2", "use-az3", "use-az5"]`).

The `customer_managed_key` block supports:
* `resource_name` - The resource name of the customer managed key as defined by the cloud provider, e.g. projects/PROJECT_ID/locations/LOCATION/keyRings/KEY_RING/cryptoKeys/KEY_NAME

The `maintenance_windows` object has these attributes:

* `mode` - Either `automatic` (Redis specified) or `manual` (User specified)
* `window` - A list of windows (if manual mode)

The `window` object has these attributes:

* `start_hour` - What hour in the day (0-23) the window opens
* `duration_in_hours` - How long the window is open (4-24 hours)
* `days` - A list of weekdays on which the window is open ('Monday', 'Tuesday' etc)

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 mins) Used when creating the subscription
* `update` - (Defaults to 30 mins) Used when updating the subscription
* `delete` - (Defaults to 10 mins) Used when destroying the subscription

## Attribute reference

* `customer_managed_key_redis_service_account` - Outputs the id of the service account associated with the subscription. Useful as part of the CMK flow.

The `region` block has these attributes:

* `networks` - List of generated network configuration

The `networks` block has these attributes:

* `networking_subnet_id` - The subnet that the subscription deploys into
* `networking_deployment_cidr` - Deployment CIDR mask for the generated
* `networking_vpc_id` - VPC id for the generated network

The `pricing` object has these attributes:

* `type` - The type of cost. E.g. 'Shards'.
* `typeDetails` - Further detail E.g. 'micro'.
* `quantity` - Self-explanatory.
* `quantityMeasurement` - Self-explanatory.
* `pricePerUnit` - Price per Unit.
* `priceCurrency` - The price currency
* `pricePeriod` - Price period. E.g. 'hour'.

## Import

`rediscloud_subscription` can be imported using the ID of the subscription, e.g.

```
$ terraform import rediscloud_subscription.subscription-resource 12345678
```
~> **Note:** the payment_method property and creation_plan block will be ignored during imports.

~> **Note:** when importing an existing Subscription, upon providing a `redis_version`, Terraform will always try to
recreate the resource. The API doesn't return this value, so we can't detect changes between states.
