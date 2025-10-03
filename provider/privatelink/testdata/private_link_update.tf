locals {
  rediscloud_subscription_name = "%s"
  rediscloud_cloud_account = "%s"
  rediscloud_private_link_share_name = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = local.rediscloud_cloud_account
}

resource "rediscloud_subscription" "pro_subscription" {
  name              = local.rediscloud_subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    dataset_size_in_gb           = 15
    quantity                     = 1
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 20000
  }
}

resource "rediscloud_private_link" "private_link" {
  subscription_id = rediscloud_subscription.pro_subscription.id
  share_name = local.rediscloud_private_link_share_name

  # principal {
  #   principal = local.rediscloud_principal_1
  #   principal_type = "aws_account"
  #   principal_alias = "terraform test aws account"
  # }

  principal {
    principal = "234567890123"
    principal_type = "aws_account"
    principal_alias = "terraform test aws account 2"
  }
}

data "rediscloud_private_link" "private_link" {
  subscription_id = rediscloud_private_link.private_link.subscription_id
}
