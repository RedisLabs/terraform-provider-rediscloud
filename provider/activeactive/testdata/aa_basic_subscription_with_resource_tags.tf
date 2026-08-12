variable "rediscloud_subscription_name" {
  type = string
}

variable "rediscloud_subscription_resource_tags" {
  type = map(string)
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name              = var.rediscloud_subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "AWS"

  creation_plan {
    dataset_size_in_gb = 1
    quantity           = 1
    region {
      region                      = "us-east-1"
      networking_deployment_cidr  = "10.0.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
  resource_tags = var.rediscloud_subscription_resource_tags
}
