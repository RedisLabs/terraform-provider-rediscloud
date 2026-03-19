variable "subscription_name" {
  type = string
}

variable "password" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_subscription" "example" {
  name                   = var.subscription_name
  payment_method_id      = data.rediscloud_payment_method.card.id
  public_endpoint_access = false

  cloud_provider {
    provider = "AWS"
    region {
      region                     = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
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
  name                         = var.subscription_name
  protocol                     = "redis"
  dataset_size_in_gb           = 1
  data_persistence             = "none"
  data_eviction                = "allkeys-random"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 1000
  enable_passwordless          = true
  password                     = var.password
  enable_default_user          = true
  replication                  = false
}
