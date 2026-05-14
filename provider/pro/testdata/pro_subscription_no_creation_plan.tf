variable "rediscloud_cloud_account" {
  type = string
}

variable "rediscloud_subscription_name" {
  type = string
}

variable "memory_storage" {
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
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = var.memory_storage

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
}
