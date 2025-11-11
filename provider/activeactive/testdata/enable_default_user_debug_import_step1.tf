# DEBUG VERSION - Imports existing database
# Template signature: fmt.Sprintf(template, subscription_id, password)
locals {
  subscription_id = "%s"
  password        = "%s"
}

# Step 1: global=true, both regions inherit (NO enable_default_user in override_region)
resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id = local.subscription_id
  name            = "matt-database-debug-testing"
  memory_limit_in_gb = 1

  # Global enable_default_user is true
  global_enable_default_user = true
  global_password            = local.password

  # Both regions inherit from global - NO enable_default_user specified
  override_region {
    name = "us-east-1"
  }

  override_region {
    name = "us-east-2"
  }

  lifecycle {
    ignore_changes = [
      # Ignore changes to fields we're not testing
      memory_limit_in_gb,
      global_data_persistence,
      global_source_ips,
      global_alert,
      global_modules,
      port,
      replication,
      throughput_measurement_by,
      throughput_measurement_value,
    ]
  }
}
