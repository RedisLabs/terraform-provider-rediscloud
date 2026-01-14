locals {
  subscription_name = "__SUBSCRIPTION_NAME__"
  database_name     = "__DATABASE_NAME__"
  password          = "__PASSWORD__"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "test" {
  name              = local.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    dataset_size_in_gb = 1
    quantity           = 1
    region {
      region                      = "us-east-1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "us-east-2"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "test" {
  subscription_id            = rediscloud_active_active_subscription.test.id
  name                       = local.database_name
  dataset_size_in_gb         = 1
  global_password            = local.password
  global_enable_default_user = false

  # us-east-1: explicitly enable (override global false)
  override_region {
    name                = "us-east-1"
    enable_default_user = true
  }

  # us-east-2: inherit global=false
  override_region {
    name = "us-east-2"
  }
}
