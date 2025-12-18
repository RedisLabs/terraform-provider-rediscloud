locals {
  subscription_name = "%s"
  database_name     = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_essentials_plan" "example" {
  name           = "Single-Zone_1GB"
  cloud_provider = "AWS"
  region         = "us-east-1"
}

data "rediscloud_essentials_database" "example" {
  subscription_id = rediscloud_essentials_subscription.example.id
  name            = rediscloud_essentials_database.example.name
}

resource "rediscloud_essentials_subscription" "example" {
  name              = local.subscription_name
  plan_id           = data.rediscloud_essentials_plan.example.id
  payment_method_id = data.rediscloud_payment_method.card.id
}

resource "rediscloud_essentials_database" "example" {
  subscription_id     = rediscloud_essentials_subscription.example.id
  name                = local.database_name
  enable_default_user = true
  password            = "j43589rhe39f"

  data_persistence = "none"
  replication      = false

  alert {
    name  = "throughput-higher-than"
    value = 80
  }
}
