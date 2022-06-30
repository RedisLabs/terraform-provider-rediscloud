# Migration guide

This guide is for the users who want to migrate their old Terraform configurations (`RedisLabs/rediscloud` `< v1.0.0`)
to the latest version.

The migration is safe, simple and will not alter any of the existing resources in your infrastructure.
The process is as follows:

1. Update your HCL files to use the latest version of the schemas for your subscriptions and databases.
2. Import your resources into a new Terraform state file.
3. Verify that the resources are imported correctly.

## Why would I want to migrate?

The version `>= 1.0.0` of the provider contains breaking changes in the schemas.
However, those changes help to improve the user experience and database management.
Those enhancements are described below:

* **Fixed the misleading plan**: In the old versions of the provider, the databases could be created in the
  `database` block inside the subscription resource. A `TypeSet` was used by the `database` attribute where an index
  value of the block is calculated by the hash of the attributes. That means, if you change an attribute inside the
  block, then Terraform would produce the misleading plan telling you that the whole database is going to be recreated.
  However, under the hood, the provider doesn't delete the database and just updates the attributes unless the `name`
  attribute was changed. In order to fix this, the database block has been moved to a separate resource.
* **Separate database resource**: In order to fix the misleading plan, the database block has been moved to a separate
  resource. This allows the user to take greater control over the database resource. That is:
  * easier access to the attributes of the database,
  * importing specific databases in the state,
  * simpler database management through the provider.

## Prerequisites:

* The RedisCloud provider `>= 1.0.0`.
* Backup your Terraform state: Make sure you have a backup of your state before you start the migration.
* Access to the Redis Cloud API: Make sure that the provider can connect to RedisCloud.
* Empty state file: Make sure to run the migration with an empty state file. You can do that by creating a new directory
  with your Terraform configuration files.

## Run migration

The `rediscloud_subscription` no longer supports the `database` block, and a new block called `creation_plan` has been
introduced. In this case, you only need to modify your existing `rediscloud_subscription` schema and create a new
resource called `rediscloud_database` for each of your databases in the subscription.

**Note**: If you want to create a new subscription, then the `creation_plan` block is required.

Here is an example of an old Terraform configuration:

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

// This resource needs to be updated to the new schema
resource "rediscloud_subscription" "example" {

  name              = "example"
  payment_method    = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  cloud_provider { ... }

  // This block will be migrated to a separate resource
  database {
    name                         = "tf-example-database"
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    password                     = "encrypted-password-0"

    alert {
      name  = "dataset-size"
      value = 40
    }
  }
}
```

To use the latest schema, you need to modify the `rediscloud_subscription` resource and add new `rediscloud_database`
resources for your databases. Like so:

* New configuration (>= `1.0.0`):

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
  
    // Add this block if you want to create a new subscription. 
    // Skip, if you are importing an existing subscription.
    // In this block, define your hardware specification for you databases in the cluster.
    creation_plan {
      average_item_size_in_bytes = 1
      memory_limit_in_gb = 1
      quantity = 1
      replication=false
      support_oss_cluster_api=false
      throughput_measurement_by = "operations-per-second"
      throughput_measurement_value = 10000
      modules = ["RedisJSON", "RedisBloom"]
  }

  // The database block has been moved to a separate resource - rediscloud_database.
  // The attributes of the database are the same as the ones in the database block in the old subscription resource schema. 
  // With the exception of the `subscription_id` attribute.
  resource "rediscloud_database" "first_database" {
      // Attach the database to the subscription.
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
   **OPTIONAL**: If you have other resources like `rediscloud_cloud_account` or `rediscloud_subscription_peering`, then
   you can import them as well:
     ```bash
     # Import rediscloud_cloud_account
     terraform import rediscloud_cloud_account.cloud_example <cloud account id>;
     # Import the peering resource. The last argument contains the subscription id and the peering id separated by a slash.
     terraform import rediscloud_subscription_peering.peering_example <subscription_id>/<cloud account id>;
     ```


4. Verify that the new state file is valid:
    ```bash
    # List the resources in the state file
    terraform state list;
    # Check if the resources are valid
    terraform state show rediscloud_subscription.example;
    terraform state show rediscloud_database.first_database;
    ```
   **OPTIONAL**: If you have other resources like `rediscloud_cloud_account` or `rediscloud_subscription_peering`, then
   you can check if they are valid:
     ```bash
     # Check if the cloud account resource is valid
     terraform state show rediscloud_cloud_account.cloud_example;
     # Check if the peering resource is valid
     terraform state show rediscloud_subscription_peering.peering_example;
     ```

Congratulations! You have successfully migrated your Terraform configuration to the new schema.
