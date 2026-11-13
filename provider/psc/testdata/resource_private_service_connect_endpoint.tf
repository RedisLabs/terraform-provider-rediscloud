variable "subscription_name" {
  type = string
}

variable "gcp_project_id" {
  type = string
}

variable "gcp_vpc_name" {
  type = string
}

variable "gcp_vpc_subnet_name" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_subscription" "subscription_resource" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    provider = "GCP"
    region {
      region                     = "us-central1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = 1
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 20000
  }
}

resource "rediscloud_private_service_connect" "psc" {
  subscription_id = rediscloud_subscription.subscription_resource.id
}

resource "rediscloud_private_service_connect_endpoint" "psce" {
  subscription_id                    = rediscloud_subscription.subscription_resource.id
  private_service_connect_service_id = rediscloud_private_service_connect.psc.private_service_connect_service_id
  gcp_project_id                     = var.gcp_project_id
  gcp_vpc_name                       = var.gcp_vpc_name
  gcp_vpc_subnet_name                = var.gcp_vpc_subnet_name
  endpoint_connection_name           = "redis-${rediscloud_subscription.subscription_resource.id}"
}
