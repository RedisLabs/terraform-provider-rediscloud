variable "subscription_name" {
  type = string
}

data "rediscloud_essentials_plan" "example" {
  name           = "30MB"
  cloud_provider = "AWS"
  region         = "us-east-1"
}

resource "rediscloud_essentials_subscription" "example" {
  name    = var.subscription_name
  plan_id = data.rediscloud_essentials_plan.example.id
}
