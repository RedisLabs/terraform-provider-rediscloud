variable "cloud_account_name" {
  type = string
}

variable "subscription_name" {
  type = string
}

variable "dataset_size_in_gb" {
  type = number
}

variable "password" {
  type = string
}

variable "ssl_certificate" {
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
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    modules                      = []
  }
}

resource "rediscloud_subscription_database" "example" {
  subscription_id              = rediscloud_subscription.example.id
  name                         = "tf-database"
  protocol                     = "redis"
  dataset_size_in_gb           = var.dataset_size_in_gb
  support_oss_cluster_api      = true
  data_persistence             = "none"
  replication                  = false
  throughput_measurement_by    = "operations-per-second"
  password                     = var.password
  throughput_measurement_value = 10000
  source_ips                   = ["10.0.0.0/8"]
  client_ssl_certificate       = var.ssl_certificate
}
