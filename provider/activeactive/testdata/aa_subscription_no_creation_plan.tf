variable "subscription_name" {
  type = string
}

variable "cloud_provider" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = var.cloud_provider

  maintenance_windows {
    mode = "automatic"
  }
}

data "rediscloud_active_active_subscription" "example" {
  name = rediscloud_active_active_subscription.example.name
}
