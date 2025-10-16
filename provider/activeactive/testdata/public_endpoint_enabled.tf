locals {
  rediscloud_subscription_name = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name                   = local.rediscloud_subscription_name
  payment_method_id      = data.rediscloud_payment_method.card.id
  cloud_provider         = "AWS"
  public_endpoint_access = true

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
}
