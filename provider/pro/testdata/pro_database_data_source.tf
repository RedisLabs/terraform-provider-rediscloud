locals {
  rediscloud_cloud_account = "%s"
  rediscloud_subscription_name = "%s"
  rediscloud_password = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = local.rediscloud_cloud_account
}
resource "rediscloud_subscription" "example" {
  name = local.rediscloud_subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }
  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=true
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    query_performance_factor	 = "2x"
    modules = ["RediSearch"]
  }
}
resource "rediscloud_subscription_database" "example" {
  subscription_id              = rediscloud_subscription.example.id
  name                         = "tf-database"
  protocol                     = "redis"
  memory_limit_in_gb           = 1
  data_persistence             = "none"
  throughput_measurement_by    = "operations-per-second"
  throughput_measurement_value = 1000
  password                     = local.rediscloud_password
  support_oss_cluster_api	     = true
  replication				     = false
  enable_default_user 		 = true
  query_performance_factor	 = "2x"
  redis_version = "7.2"
  modules = [
    {
      name: "RediSearch"
    }
  ]
}

data "rediscloud_database" "example-by-id" {
  subscription_id = rediscloud_subscription.example.id
  db_id = rediscloud_subscription_database.example.db_id
}

data "rediscloud_database" "example-by-name" {
  subscription_id = rediscloud_subscription.example.id
  name = rediscloud_subscription_database.example.name
}
