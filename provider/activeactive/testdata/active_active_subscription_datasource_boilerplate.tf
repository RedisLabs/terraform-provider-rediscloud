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
