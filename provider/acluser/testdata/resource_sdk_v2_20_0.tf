terraform {
  required_providers {
    rediscloud = {
      source  = "RedisLabs/rediscloud"
      version = "2.20.0"
    }
  }
}

provider "rediscloud" {}

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

variable "database_password" {
  type      = string
  sensitive = true
}

variable "role_name" {
  type = string
}

variable "user_name" {
  type = string
}

variable "user_password" {
  type      = string
  sensitive = true
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
  cloud_provider {
    resource_tags = {
      team  = var.team
      owner = var.owner
    }

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
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
  }
}

resource "rediscloud_subscription_database" "example" {
  subscription_id              = rediscloud_subscription.example.id
  name                         = "example"
  protocol                     = "redis"
  dataset_size_in_gb           = 1
  data_persistence             = "none"
  data_eviction                = "allkeys-random"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 1000
  password                     = var.database_password
  replication                  = false
  enable_default_user          = true
  redis_version                = "7.4"
}

resource "rediscloud_acl_role" "example" {
  name = var.role_name
  rule {
    name = "Read-Only"
    database {
      subscription = rediscloud_subscription.example.id
      database     = rediscloud_subscription_database.example.db_id
    }
  }
}

resource "rediscloud_acl_user" "test" {
  name     = var.user_name
  role     = rediscloud_acl_role.example.name
  password = var.user_password
}
