locals {
  rediscloud_cloud_account     = "__CLOUD_ACCOUNT__"
  rediscloud_subscription_name = "__SUBSCRIPTION_NAME__"
  auto_minor_version_upgrade   = "__AUTO_MINOR_VERSION_UPGRADE__" == "true"
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
    memory_limit_in_gb           = 1
    quantity                     = 1
    replication                  = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
  }
}

resource "rediscloud_subscription_database" "example" {
  subscription_id              = rediscloud_subscription.example.id
  name                         = "auto-minor-version-upgrade-test"
  protocol                     = "redis"
  memory_limit_in_gb           = 1
  data_persistence             = "none"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 1000
  auto_minor_version_upgrade   = local.auto_minor_version_upgrade
}