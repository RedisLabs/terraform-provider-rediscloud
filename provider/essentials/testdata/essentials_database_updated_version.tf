locals {
  subscription_name = "%s"
  database_name     = "%s"
  redis_version     = "%s"
  password          = "%s"
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

resource "rediscloud_essentials_subscription" "example" {
  name              = local.subscription_name
  plan_id           = data.rediscloud_essentials_plan.example.id
  payment_method_id = data.rediscloud_payment_method.card.id
}

resource "rediscloud_essentials_database" "example" {
  subscription_id  = rediscloud_essentials_subscription.example.id
  name             = local.database_name
  protocol         = "redis"
  redis_version    = local.redis_version
  replication      = false
  data_persistence = "aof-every-write"

  password = local.password

  alert {
    name  = "throughput-higher-than"
    value = 85
  }
}
