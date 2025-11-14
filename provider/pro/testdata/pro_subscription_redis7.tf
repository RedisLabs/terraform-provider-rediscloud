locals {
  cloud_account_name = "%s"
  subscription_name  = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = local.cloud_account_name
}

resource "rediscloud_subscription" "example" {

  name              = local.subscription_name
  payment_method    = "credit-card"
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
    query_performance_factor     = "4x"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    modules                      = ["RedisJSON", "RedisBloom", "RediSearch"]
  }
}
