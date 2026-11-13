variable "team" {
  type = string
}

variable "owner" {
  type = string
}

variable "cloud_account_name" {
  type = string
}

variable "subscription_name" {
  type = string
}

variable "subscription_cidr" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "aws_account_id" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "vpc_cidr" {
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

resource "rediscloud_subscription" "example" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  cloud_provider {
    resource_tags = {
      team  = var.team
      owner = var.owner
    }

    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "eu-west-1"
      networking_deployment_cidr   = var.subscription_cidr
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    memory_limit_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
    modules                      = []
  }
}

resource "rediscloud_subscription_peering" "test" {
  subscription_id = rediscloud_subscription.example.id
  region          = var.aws_region
  aws_account_id  = var.aws_account_id
  vpc_id          = var.vpc_id
  vpc_cidr        = var.vpc_cidr
}

data "rediscloud_subscription_peerings" "example" {
  subscription_id = rediscloud_subscription_peering.test.subscription_id
}
