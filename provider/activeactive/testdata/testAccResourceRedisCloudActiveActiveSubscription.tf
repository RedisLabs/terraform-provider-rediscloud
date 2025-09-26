locals {
  rediscloud_subscription_name = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name              = local.rediscloud_subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    modules            = ["RedisJSON"]
    quantity           = 1
    region {
      region                      = "us-east-1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "us-east-2"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }

  maintenance_windows {
    mode = "manual"
    window {
      start_hour        = 22
      duration_in_hours = 8
      days              = ["Monday", "Thursday"]
    }
    window {
      start_hour        = 12
      duration_in_hours = 6
      days              = ["Friday", "Saturday", "Sunday"]
    }
  }
}

resource "rediscloud_active_active_subscription_database" "example" {
  subscription_id         = rediscloud_active_active_subscription.example.id
  name                    = local.rediscloud_subscription_name
  dataset_size_in_gb      = 1
  global_data_persistence = "aof-every-1-second"
  global_password         = "some-random-pass-2"
  global_source_ips       = ["192.168.0.0/16"]
  global_alert {
    name  = "dataset-size"
    value = 40
  }

  global_modules = ["RedisJSON"]



  override_region {
    name                       = "us-east-2"
    enable_default_user = true
    override_global_source_ips = ["192.10.0.0/16"]
  }

  override_region {
    name                             = "us-east-1"
    override_global_data_persistence = "none"
    override_global_password         = "region-specific-password"
    override_global_alert {
      name  = "dataset-size"
      value = 60
    }
  }

  tags = {
    "environment" = "production"
    "cost_center" = "0700"
  }
}


data "rediscloud_active_active_subscription" "example" {
  name = rediscloud_active_active_subscription.example.name
}

data "rediscloud_active_active_subscription_regions" "example" {
  subscription_name = rediscloud_active_active_subscription.example.name
}

