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

resource "rediscloud_active_active_subscription" "example" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "GCP"
  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "europe-west1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "europe-west2"
      networking_deployment_cidr  = "192.168.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_peering" "test" {
  subscription_id  = rediscloud_active_active_subscription.example.id
  provider_name    = "GCP"
  gcp_project_id   = var.gcp_project_id
  gcp_network_name = var.gcp_network_name

  source_region = "europe-west2"
}
