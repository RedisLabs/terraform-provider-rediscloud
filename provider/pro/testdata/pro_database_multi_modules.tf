locals {
  rediscloud_cloud_account     = "%s"
  rediscloud_subscription_name = "%s"
  database_name                = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = local.rediscloud_cloud_account
}

resource "rediscloud_subscription" "example" {
  name              = local.rediscloud_subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"
  redis_version     = "7.4"
  allowlist {
    cidrs              = ["192.168.0.0/16"]
    security_group_ids = []
  }
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
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
    modules                      = ["RedisJSON", "RedisBloom"]
  }
}

resource "rediscloud_subscription_database" "example" {
  subscription_id              = rediscloud_subscription.example.id
  name                         = local.database_name
  protocol                     = "redis"
  dataset_size_in_gb           = 1
  data_persistence             = "none"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 1000
  modules = [
    {
      name = "RedisJSON"
    },
    {
      name = "RedisBloom"
    }
  ]
  alert {
    name  = "latency"
    value = 11
  }
}
