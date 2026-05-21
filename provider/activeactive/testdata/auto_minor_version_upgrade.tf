variable "subscription_name" {
  type = string
}

variable "database_name" {
  type = string
}

variable "auto_minor_version_upgrade" {
  type = bool
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

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id            = rediscloud_active_active_subscription.example.id
  name                       = var.database_name
  dataset_size_in_gb         = 1
  data_eviction              = "allkeys-random"
  auto_minor_version_upgrade = var.auto_minor_version_upgrade

  override_region {
    name = "us-east-1"
  }

  override_region {
    name = "us-east-2"
  }
}
