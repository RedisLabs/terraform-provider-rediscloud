variable "subscription_name" {
  type = string
}

variable "database_name" {
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

resource "rediscloud_subscription_database" "database_resource" {
  name                         = var.database_name
  subscription_id              = rediscloud_subscription.subscription_resource.id
  dataset_size_in_gb           = 1
  replication                  = true
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 20000
}

resource "rediscloud_private_service_connect" "psc" {
  subscription_id = rediscloud_subscription.subscription_resource.id
  depends_on      = [rediscloud_subscription_database.database_resource]
}

data "rediscloud_private_service_connect" "psc" {
  subscription_id = rediscloud_private_service_connect.psc.subscription_id
}
