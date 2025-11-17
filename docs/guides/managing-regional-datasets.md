### Managing Dataset Size with the Regions Resource

For Active-Active databases, you can manage `dataset_size_in_gb` via either the `rediscloud_active_active_database` resource or the `rediscloud_active_active_subscription_regions` resource. Managing it via the regions resource allows you to update both database sizing and per-region throughput in a single operation.

~> **Critical:** The `dataset_size_in_gb` field is a global property that applies to all regional instances of this Active-Active database. You must choose **ONE** of the patterns below and use it consistently. **Never set different values in both resources** - this will cause Terraform to continuously revert changes between apply operations.

#### Option 1: Reference Pattern (Recommended)

Set `dataset_size_in_gb` on the regions resource and reference it from the database resource. This ensures a single source of truth and prevents conflicts:

```hcl
resource "rediscloud_active_active_subscription_regions" "regions-resource" {
    subscription_id = rediscloud_active_active_subscription.subscription-resource.id
    dataset_size_in_gb = 10

    region {
      region = "us-east-1"
      networking_deployment_cidr = "192.168.0.0/24"
      database {
          database_id = rediscloud_active_active_subscription_database.database-resource.db_id
          database_name = rediscloud_active_active_subscription_database.database-resource.name
          local_write_operations_per_second = 1000
          local_read_operations_per_second = 1000
      }
    }
}

resource "rediscloud_active_active_subscription_database" "database-resource" {
    subscription_id = rediscloud_active_active_subscription.subscription-resource.id
    name = "database-name"
    # Reference the regions resource to avoid conflicts
    dataset_size_in_gb = rediscloud_active_active_subscription_regions.regions-resource.dataset_size_in_gb
    global_data_persistence = "aof-every-1-second"
}
```

#### Option 2: Explicit Dependency Pattern

Alternatively, set the value in both resources but use `depends_on` to ensure proper ordering:

```hcl
resource "rediscloud_active_active_subscription_regions" "regions-resource" {
    subscription_id = rediscloud_active_active_subscription.subscription-resource.id
    dataset_size_in_gb = 10

    region {
      region = "us-east-1"
      networking_deployment_cidr = "192.168.0.0/24"
      database {
          database_id = rediscloud_active_active_subscription_database.database-resource.db_id
          database_name = rediscloud_active_active_subscription_database.database-resource.name
          local_write_operations_per_second = 1000
          local_read_operations_per_second = 1000
      }
    }
}

resource "rediscloud_active_active_subscription_database" "database-resource" {
    subscription_id = rediscloud_active_active_subscription.subscription-resource.id
    name = "database-name"
    dataset_size_in_gb = 10  # Must match the value in regions resource
    global_data_persistence = "aof-every-1-second"

    # Ensure regions resource updates first
    depends_on = [rediscloud_active_active_subscription_regions.regions-resource]
}
```

-> **Note:** Option 1 (reference pattern) is recommended as it eliminates the risk of setting different values and provides clearer intent.

#### Option 3: Database Resource Only

If you don't need to coordinate dataset size changes with per-region throughput updates, you can manage `dataset_size_in_gb` solely on the database resource without using the regions resource:

```hcl
resource "rediscloud_active_active_subscription_database" "database-resource" {
    subscription_id = rediscloud_active_active_subscription.subscription-resource.id
    name = "database-name"
    dataset_size_in_gb = 10
    global_data_persistence = "aof-every-1-second"
}
```
