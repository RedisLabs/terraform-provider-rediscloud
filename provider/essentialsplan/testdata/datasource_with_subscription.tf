variable "card_type" {
  type = string
}

variable "last_four_numbers" {
  type = string
}

variable "plan_name" {
  type = string
}

variable "cloud_provider" {
  type = string
}

variable "region" {
  type = string
}

variable "subscription_name" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = var.card_type
  last_four_numbers = var.last_four_numbers
}

data "rediscloud_essentials_plan" "fixed" {
  name           = var.plan_name
  cloud_provider = var.cloud_provider
  region         = var.region
}

resource "rediscloud_essentials_subscription" "fixed" {
  name              = var.subscription_name
  plan_id           = data.rediscloud_essentials_plan.fixed.id
  payment_method_id = data.rediscloud_payment_method.card.id
}

data "rediscloud_essentials_plan" "with_subscription" {
  name            = data.rediscloud_essentials_plan.fixed.name
  subscription_id = rediscloud_essentials_subscription.fixed.id
}