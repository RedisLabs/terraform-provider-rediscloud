variable "subscription_name" {
  type = string
}

variable "database_name" {
  type = string
}

variable "database_password" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "us-east-1"
      networking_deployment_cidr  = "10.0.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "us-east-2"
      networking_deployment_cidr  = "10.1.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id                       = rediscloud_active_active_subscription.example.id
  name                                  = var.database_name
  memory_limit_in_gb                    = 3
  support_oss_cluster_api               = false
  external_endpoint_for_oss_cluster_api = false

  global_data_persistence = "none"
  global_password         = var.database_password
  global_alert {
    name  = "dataset-size"
    value = 1
  }
}

data "rediscloud_active_active_subscription_regions" "example" {
  subscription_name = rediscloud_active_active_subscription.example.name
}

resource "rediscloud_active_active_subscription_regions" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  delete_regions  = false
  region {
    region                     = "us-east-1"
    networking_deployment_cidr = "10.0.0.0/24"
    recreate_region            = false
    database {
      database_id                       = rediscloud_active_active_subscription_database.example.db_id
      database_name                     = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second  = 1000
    }
  }
  region {
    region                     = "us-east-2"
    networking_deployment_cidr = "10.1.0.0/24"
    recreate_region            = false
    database {
      database_id                       = rediscloud_active_active_subscription_database.example.db_id
      database_name                     = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second  = 1000
    }
  }
  region {
    region                     = "eu-west-2"
    networking_deployment_cidr = "10.3.0.0/24"
    recreate_region            = false
    database {
      database_id                       = rediscloud_active_active_subscription_database.example.db_id
      database_name                     = rediscloud_active_active_subscription_database.example.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second  = 1000
    }
  }
}
