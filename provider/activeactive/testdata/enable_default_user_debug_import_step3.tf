# DEBUG VERSION - Imports existing database
# Template signature: fmt.Sprintf(template, subscription_id, password)
locals {
  subscription_id = "%s"
  password        = "%s"
}

# Step 3: global=false, us-east-1 explicit true
resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id = local.subscription_id
  name            = "matt-test-debugging"
  memory_limit_in_gb = 1

  # Global enable_default_user is false
  global_enable_default_user = false
  global_password            = local.password

  # eu-west-1 inherits from global
  override_region {
    name = "eu-west-1"
  }

  # us-east-1 explicitly set to true (differs from global)
  override_region {
    name                = "us-east-1"
    enable_default_user = true
  }

  lifecycle {
    ignore_changes = [
      # Ignore changes to fields we're not testing
      name,
      memory_limit_in_gb,
      global_data_persistence,
      global_source_ips,
      global_alert,
      global_modules,
      port,
    ]
  }
}
