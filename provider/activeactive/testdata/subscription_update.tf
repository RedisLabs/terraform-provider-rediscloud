locals {
  rediscloud_subscription_name   = "%s"
  rediscloud_cloud_provider_name = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name              = local.rediscloud_subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = local.rediscloud_cloud_provider_name

  maintenance_windows {
    mode = "automatic"
  }
}

data "rediscloud_active_active_subscription" "example" {
  name = rediscloud_active_active_subscription.example.name
}

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id         = rediscloud_active_active_subscription.example.id
  name                    = local.rediscloud_subscription_name
  dataset_size_in_gb      = 1
  global_data_persistence = "aof-every-1-second"
  global_password         = "some-random-pass-2"
  global_source_ips = ["192.168.0.0/16"]
  global_alert {
    name  = "dataset-size"
    value = 40
  }

  global_modules = ["RedisJSON"]
  global_enable_default_user = false

  override_region {
    name                             = "us-east-1"
    override_global_data_persistence = "none"
    override_global_password         = "region-specific-password"
    override_global_alert {
      name  = "dataset-size"
      value = 60
    }
  }

  override_region {
    name                = "us-east-2"
    enable_default_user = false
    override_global_source_ips = ["192.10.0.0/16"]
  }

  override_region {
    name                = "us-east-3"
    enable_default_user = true
  }

  tags = {
    "environment" = "production"
    "cost_center" = "0700"
  }
}
