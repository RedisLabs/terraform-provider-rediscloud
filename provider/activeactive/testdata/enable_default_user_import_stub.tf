# Minimal stub config for import - just defines the resource shell
# Template signature: fmt.Sprintf(template, subscription_id, password)
locals {
  subscription_id = "%s"
  password        = "%s"
}

# Minimal resource definition for import
resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id = local.subscription_id
  name            = "matt-test-debugging"
  memory_limit_in_gb = 1

  global_enable_default_user = true
  global_password            = local.password

  override_region {
    name = "eu-west-1"
  }

  override_region {
    name = "us-east-1"
  }
}
