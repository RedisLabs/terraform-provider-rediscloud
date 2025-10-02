locals {
  rediscloud_subscription_name = "%s"
}

data "rediscloud_subscription" "example" {
    name = local.rediscloud_subscription_name
}
