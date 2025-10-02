locals {
  rediscloud_subscription_name = "%s"
}


resource "rediscloud_subscription_database" "example" {
  subscription_id                       = rediscloud_subscription.example.id
  name                                  = local.rediscloud_subscription_name
  protocol                              = "redis"
  dataset_size_in_gb                    = 3
  data_persistence                      = "none"
  data_eviction                         = "allkeys-random"
  throughput_measurement_by             = "operations-per-second"
  throughput_measurement_value          = 1000
  password                              = "%s"
  support_oss_cluster_api               = false
  external_endpoint_for_oss_cluster_api = false
  replication                           = false
  average_item_size_in_bytes            = 0
  client_ssl_certificate                = ""
  periodic_backup_path                  = ""
  enable_default_user                   = true
  redis_version                         = 8.0

  alert {
    name  = "dataset-size"
    value = 1
  }

  modules = [
    {
      name = "RedisBloom"
    }
  ]

  tags = {
    "market"   = "emea"
    "material" = "cardboard"
  }
}
