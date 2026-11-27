variable "region" { default = "us-east-1" }
variable "plan_name" { default = "30MB" }
variable "payment_method_id" {
  type    = number
  default = null
}
variable "redis_version" {
  type    = string
  default = null
}

data "rediscloud_essentials_plan" "test" {
  name           = var.plan_name
  cloud_provider = "AWS"
  region         = var.region
}

resource "rediscloud_essentials_subscription" "test" {
  name              = var.name
  plan_id           = data.rediscloud_essentials_plan.test.id
  payment_method_id = var.payment_method_id
}

resource "rediscloud_essentials_database" "test" {
  subscription_id     = rediscloud_essentials_subscription.test.id
  name                = substr("${var.name}-db", 0, 40)
  protocol            = "stack"
  redis_version       = var.redis_version
  enable_default_user = true
  data_persistence    = "none"
  replication         = false
}

locals {
  database_id = tostring(rediscloud_essentials_database.test.db_id)
}
