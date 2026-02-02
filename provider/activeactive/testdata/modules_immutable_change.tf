locals {
  subscription_name = "__SUBSCRIPTION_NAME__"
  database_name     = "__DATABASE_NAME__"
  password          = "__PASSWORD__"
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
  subscription_id                       = rediscloud_active_active_subscription.example.id
  name                                  = local.database_name
  redis_version                         = "7.4"
  dataset_size_in_gb                    = 1
  support_oss_cluster_api               = false
  external_endpoint_for_oss_cluster_api = false
  enable_tls                            = false

  global_data_persistence = "none"
  global_password         = local.password
  global_source_ips       = ["192.168.0.0/16"]
  # Changed from RedisJSON to RedisBloom - this should error
  global_modules = ["RedisBloom"]

  override_region {
    name = "us-east-1"
  }
  override_region {
    name = "us-east-2"
  }
}
