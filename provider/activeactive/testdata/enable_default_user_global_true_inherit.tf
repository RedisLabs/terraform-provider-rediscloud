# Template signature: fmt.Sprintf(template, subscription_name, database_name, password)
locals {
  subscription_name = "%s"
  database_name     = "%s"
  password          = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "example" {
  name              = local.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    dataset_size_in_gb = 1
    quantity           = 1

    region {
      region                     = "us-east-1"
      networking_deployment_cidr = "10.0.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }

    region {
      region                     = "us-east-2"
      networking_deployment_cidr = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id    = rediscloud_active_active_subscription.example.id
  name               = local.database_name
  memory_limit_in_gb = 1

  # Global enable_default_user is true (default behavior)
  global_enable_default_user = true
  global_password            = local.password

  # Both regions inherit from global - NO enable_default_user specified
  override_region {
    name = "us-east-1"
  }

  override_region {
    name = "us-east-2"
  }
}
