variable "subscription_name" {
  type = string
}

variable "gcp_project_id" {
  type = string
}

variable "gcp_network_name" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_subscription" "example" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  cloud_provider {
    provider         = "GCP"
    cloud_account_id = 1
    region {
      region                       = "europe-west1"
      networking_deployment_cidr   = "192.168.0.0/24"
      preferred_availability_zones = []
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
  subscription_id  = rediscloud_subscription.example.id
  provider_name    = "GCP"
  gcp_project_id   = var.gcp_project_id
  gcp_network_name = var.gcp_network_name
}
