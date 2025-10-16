locals {
  rediscloud_subscription_name = "%s"
  rediscloud_database_password = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_subscription" "example" {
  name                   = local.rediscloud_subscription_name
  payment_method_id      = data.rediscloud_payment_method.card.id
  public_endpoint_access = true

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
  subscription_id                       = rediscloud_subscription.example.id
  name                                  = "example"
  protocol                              = "redis"
  dataset_size_in_gb                    = 1
  data_persistence                      = "none"
  data_eviction                         = "allkeys-random"
  throughput_measurement_by             = "operations-per-second"
  throughput_measurement_value          = 1000
  password                              = local.rediscloud_database_password
  support_oss_cluster_api               = false
  external_endpoint_for_oss_cluster_api = false
  replication                           = false
  average_item_size_in_bytes            = 0
  client_ssl_certificate                = ""
  enable_default_user                   = true
  redis_version                         = "8.2"

  alert {
    name  = "dataset-size"
    value = 1
  }
}

output "db_source_ips" {
  value = rediscloud_subscription_database.example.source_ips
}
