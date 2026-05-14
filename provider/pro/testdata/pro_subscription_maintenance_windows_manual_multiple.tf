variable "rediscloud_cloud_account" {
  type = string
}

variable "rediscloud_subscription_name" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = var.rediscloud_cloud_account
}

resource "rediscloud_subscription" "example" {
  name              = var.rediscloud_subscription_name
  payment_method    = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "eu-west-1"
      networking_deployment_cidr   = "10.0.24.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    modules                      = ["RedisJSON", "RedisBloom"]
  }

  maintenance_windows {
    mode = "manual"
    window {
      start_hour        = 22
      duration_in_hours = 8
      days              = ["Monday", "Thursday"]
    }
    window {
      start_hour        = 12
      duration_in_hours = 6
      days              = ["Friday", "Saturday", "Sunday"]
    }
  }
}

data "rediscloud_subscription" "example" {
  name = rediscloud_subscription.example.name
}
