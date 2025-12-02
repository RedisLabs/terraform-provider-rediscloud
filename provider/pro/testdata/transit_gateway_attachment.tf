locals {
  cloud_account_name  = "%s"
  subscription_name   = "%s"
  aws_tgw_uid         = "%s"
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

  allowlist {
    cidrs              = ["192.168.0.0/16"]
    security_group_ids = []
  }

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "us-east-1"
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = ["us-east-1a"]
    }
  }

  creation_plan {
    memory_limit_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    modules                      = []
  }
}

data "rediscloud_transit_gateway" "test" {
  subscription_id = rediscloud_subscription.example.id
  aws_tgw_uid     = local.aws_tgw_uid
}

resource "rediscloud_transit_gateway_attachment" "test" {
  subscription_id = rediscloud_subscription.example.id
  tgw_id          = data.rediscloud_transit_gateway.test.tgw_id
}
