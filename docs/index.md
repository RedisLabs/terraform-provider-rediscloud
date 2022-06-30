---
layout: "rediscloud"
page_title: "Provider: Redis Enterprise Cloud"
description: |-
   The Redis Enterprise Cloud provider is used to interact with the resources supported by Redis Enterprise Cloud. The provider needs to be configured with the proper credentials before it can be used..
---

# Redis Enterprise Cloud Provider

The Redis Enterprise Cloud provider is used to interact with the resources supported by Redis Enterprise Cloud . The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available provider resources and data sources.

## Configure Redis Enterprise Cloud Programmatic Access

In order to setup authentication with the Redis Enterprise Cloud provider a programmatic API key must be generated for Redis Enterprise Cloud.   The [Redis Enterprise Cloud documentation](https://docs.redislabs.com/latest/rc/api/how-to/enable-your-account-to-use-api/) contains the most up-to-date instructions for creating and managing your key(s) and IP access.

## Example Usage

```hcl
provider "rediscloud" {
}

# Example resource configuration
resource "rediscloud_subscription" "example" {
  # ...
}
```

## Argument Reference

In addition to [generic `provider` arguments](https://www.terraform.io/docs/configuration/providers.html)
(e.g. `alias` and `version`), the following arguments are supported in the Redis Cloud
`provider` block:
 
* `url` - (Optional) This is the URL of Redis Enterprise Cloud and will default to `https://api.redislabs.com/v1`.
This can also be set by the `REDISCLOUD_URL` environment variable. 

* `api_key` - (Optional) This is the Redis Enterprise Cloud API key. It must be provided but can also be set by the
`REDISCLOUD_ACCESS_KEY` environment variable.

* `secret_key` - (Optional) This is the Redis Enterprise Cloud API secret key. It must be provided but can also be set
by the `REDISCLOUD_SECRET_KEY` environment variable.


# Migration guide

## Overview
This guide is for the users who want to migrate their old Terraform configurations (`RedisLabs/rediscloud` `< v1.0.0`) to the latest version.

The migration is safe, simple and will not alter any of the existing resources in your infrastructure.

## Why would I want to migrate?
The version `>= 1.0.0` of the provider contains breaking changes in the schemas.
However, those changes help to improve the user experience and database management. 
Each of them is described below:
* **Fixed the misleading plan**: In the old versions of the provider, the databases could be created in the 
`database` block inside the subscription resource. A `TypeSet` was used by the `database` attribute where an index value of the block is calculated by the hash of the attributes. That means, if you change an attribute inside the block, then Terraform would produce the misleading plan telling you that the whole database is going to be recreated. However, under the hood the provider doesn't delete the database and just updates the attributes, unless the `name` attribute was changed. In order to fix this, the database block has been moved to a separate resource. 
* **Separate database resource**: In order to fix the misleading plan, the database block has been moved to a separate resource. This allows the user to take a greater control over the database resource. That is:
  * easier access to the attributes of the database, 
  * importing specific databases in the state,
  * simpler database management through the provider.


## Prerequisites:
* The RedisCloud provider >= 1.0.0
* Backup your Terraform state: Make sure you have a backup of your state before you start the migration.
* Access to the Redis Cloud API: Make sure that the provider can connect to RedisCloud.
* Empty state file: Make sure to run the migration with an empty state file. You can do that by creating a new directory with your Terraform configuration files. 


## Run migration

Your old configuration might look like this:
```hcl
terraform {
  required_providers {
    rediscloud = {
      source  = "RedisLabs/rediscloud"
      version = "0.3.0"
    }
  }
}

data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "example" {

  name = "example"
  payment_method = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  cloud_provider {...}

  // This block will be migrated to a separate resource
  database {
    name = "tf-example-database"
    protocol = "redis"
    memory_limit_in_gb = 1
    data_persistence = "none"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    password = "encrypted-password-0"

    alert {
      name = "dataset-size"
      value = 40
    }
  }
}
```

In the new schema, the `database` block will be moved to a separate resource, and the block will be replaced by the `creation_plan` block.

### Run migration:

If you have an existing infrastructure with the old provider, you can migrate the resources to the new schemas by following the steps in this section.

In this process: 
- The `database` block will be moved to a separate resource.
- The `creation_plan` block will be ignored, since it can only be used to create a new subscription.

* New configuration (>= `1.0.0`).

  Your new configuration should look similar to this:
  ```hcl
  terraform {
    required_providers {
      rediscloud = {
        source  = "RedisLabs/rediscloud"
        version = "1.0.0"
      }
    }
  }

  data "rediscloud_payment_method" "card" {
    card_type = "Visa"
  }
  
  resource "rediscloud_subscription" "example" {
    name = "example"
    memory_storage = "ram"
    payment_method_id = data.rediscloud_payment_method.card.id
  
    cloud_provider {...}
  
  // The database block has been moved to a separate resource - rediscloud_database.
  resource "rediscloud_database" "first_database" {
      // The attributes of the database are the same as the ones in the database block in the subscription resource. 
      // With the exception of the `subscription_id` attribute.
      subscription_id = rediscloud_subscription.example.id
      name = "tf-example-database"
      protocol = "redis"
      memory_limit_in_gb = 1
      data_persistence = "none"
      throughput_measurement_by = "operations-per-second"
      throughput_measurement_value = 10000
      password = "encrypted-password-0"
  
      alert {
        name = "dataset-size"
        value = 40
      }
  }
  ```

#### Steps:

1. Create a new directory with your new Terraform configuration files.
2. Run `terraform init` to initialize the new directory.
3. Run the following commands to import the resources into the state file:
    ```bash
    # Import the subscription resource
    terraform import rediscloud_subscription.example <subscription id>;
    # Import the database resource. The last argument contains the subscription id and the database id separated by a slash.
    terraform import rediscloud_database.first_database <subscription id>/<database id>;
    ```
4. Verify that the new state file is valid:
    ```bash
    # List the resources in the state file
    terraform state list;
    # Check if the resources are valid
    terraform state show rediscloud_subscription.example;
    terraform state show rediscloud_database.first_database;
    ```

### Run migration: For a new infrastructure:

If you have old Terraform configurations, but you don't have an existing infrastructure, you can migrate the resources to the new schemas by following the steps in this section.

In this example, the `database` block will be moved to a separate resource. The `creation_plan` block will be added to define a well-optimised hardware specification for your databases in the cluster.

* New configuration (>= `1.0.0`).

  Your new configuration should look similar to this:
  ```hcl
  terraform {
    required_providers {
      rediscloud = {
        source  = "RedisLabs/rediscloud"
        version = "1.0.0"
      }
    }
  }

  data "rediscloud_payment_method" "card" {
    card_type = "Visa"
  }
  
  resource "rediscloud_subscription" "example" {
    name = "example"
    memory_storage = "ram"
    payment_method_id = data.rediscloud_payment_method.card.id
  
    cloud_provider {...}

   // This block is required for creating a new subscription.    
   creation_plan {
    average_item_size_in_bytes = 1
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
  }
  
  // The database block has been moved to a separate resource - rediscloud_database.
  resource "rediscloud_database" "first_database" {
      // The attributes of the database are the same as the ones in the database block in the subscription resource. 
      // With the exception of the `subscription_id` attribute.
      subscription_id = rediscloud_subscription.example.id
      name = "tf-example-database"
      protocol = "redis"
      memory_limit_in_gb = 1
      data_persistence = "none"
      throughput_measurement_by = "operations-per-second"
      throughput_measurement_value = 10000
      password = "encrypted-password-0"
  
      alert {
        name = "dataset-size"
        value = 40
      }
  }
  ```
