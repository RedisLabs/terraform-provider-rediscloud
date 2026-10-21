terraform {
  required_version = ">= 1.11.1"
}

variable "name" { type = string }
variable "region" { type = string }
variable "cloud_account_name" { type = string }
variable "team" { type = string }
variable "owner" { type = string }
variable "card_type" { type = string }
variable "card_last_four" { type = string }
variable "database_count" {
  type    = number
  default = 1
}

data "rediscloud_payment_method" "card" {
  card_type         = var.card_type
  last_four_numbers = var.card_last_four
}

data "rediscloud_cloud_account" "account" {
  count                    = var.cloud_account_name == "" ? 0 : 1
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = var.cloud_account_name
}

resource "rediscloud_subscription" "test" {
  name              = var.name
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  cloud_provider {
    provider         = "AWS"
    cloud_account_id = var.cloud_account_name == "" ? 1 : data.rediscloud_cloud_account.account[0].id
    resource_tags    = var.cloud_account_name == "" ? null : { team = var.team, owner = var.owner }
    region {
      region                      = var.region
      networking_deployment_cidr  = "10.0.0.0/24"
      multiple_availability_zones = true
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = var.database_count
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
  }
}

resource "rediscloud_subscription_database" "test" {
  count                        = var.database_count
  subscription_id              = rediscloud_subscription.test.id
  name                         = "${var.name}-db-${count.index}"
  dataset_size_in_gb           = 1
  redis_version                = "8.6"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 1000
  replication                  = true
  enable_tls                   = true
  enable_default_user          = true
  data_persistence             = "aof-every-1-second"
  data_eviction                = "allkeys-lru"
  source_ips                   = ["0.0.0.0/0"]
}
