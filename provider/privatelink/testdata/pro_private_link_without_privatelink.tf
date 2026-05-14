variable "subscription_name" {
  type = string
}

variable "cloud_account_name" {
  type = string
}

variable "database_password" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = var.cloud_account_name
}

resource "rediscloud_subscription" "pro_subscription" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "eu-west-1"
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = 1
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 20000
  }
}

resource "rediscloud_subscription_database" "pro_database" {
  subscription_id              = rediscloud_subscription.pro_subscription.id
  name                         = "db"
  memory_limit_in_gb           = 1
  password                     = var.database_password
  protocol                     = "redis"
  data_persistence             = "none"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 10000
}

# privatelink resource intentionally removed to test deletion
