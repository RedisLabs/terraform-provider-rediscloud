variable "subscription_name" {
  type = string
}

data "rediscloud_essentials_plan" "example" {
  name           = "30MB"
  cloud_provider = "AWS"
  region         = "us-east-1"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_essentials_subscription" "example" {
  name    = var.subscription_name
  plan_id = data.rediscloud_essentials_plan.example.id
  # payment_method = "credit-card"
  # payment_method_id = data.rediscloud_payment_method.card.id
}

data "rediscloud_essentials_subscription" "example" {
  name = rediscloud_essentials_subscription.example.name
}
