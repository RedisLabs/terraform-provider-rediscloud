locals {
  rediscloud_subscription_name = "%s"
  rediscloud_database_name     = "%s"
  redis_version                = "%s"
}

data "rediscloud_essentials_plan" "example" {
  name           = "30MB"
  cloud_provider = "AWS"
  region         = "us-east-1"
}

resource "rediscloud_essentials_subscription" "example" {
  name    = local.rediscloud_subscription_name
  plan_id = data.rediscloud_essentials_plan.example.id
}

resource "rediscloud_essentials_database" "example" {
  subscription_id     = rediscloud_essentials_subscription.example.id
  name                = local.rediscloud_database_name
  enable_default_user = true
  password            = "TestPassword123!"
  redis_version       = local.redis_version

  data_persistence = "none"
  replication      = false

  tags = {
    "environment" = "testing"
    "purpose"     = "version-upgrade"
  }
}
