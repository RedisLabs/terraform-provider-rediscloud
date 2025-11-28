locals {
  subscription_name = "%s"
  database_name     = "%s"
  dataset_size      = %f
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name              = local.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
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

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id         = rediscloud_active_active_subscription.example.id
  name                    = local.database_name
  memory_limit_in_gb      = 1
  global_data_persistence = "aof-every-1-second"
  global_password         = "some-random-pass-2"
}

resource "rediscloud_active_active_subscription_regions" "example" {
  subscription_id    = rediscloud_active_active_subscription.example.id
  delete_regions     = false
  dataset_size_in_gb = local.dataset_size

  region {
    region                       = "us-east-1"
    networking_deployment_cidr   = "192.168.0.0/24"
    database {
      database_id                        = rediscloud_active_active_subscription_database.example.db_id
      database_name                      = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second  = 1000
      local_read_operations_per_second   = 1000
    }
  }

  region {
    region                       = "us-east-2"
    networking_deployment_cidr   = "10.0.1.0/24"
    database {
      database_id                        = rediscloud_active_active_subscription_database.example.db_id
      database_name                      = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second  = 2000
      local_read_operations_per_second   = 4000
    }
  }

  depends_on = [rediscloud_active_active_subscription_database.example]
}
