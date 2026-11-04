variable "subscription_name" {
  type = string
}

variable "database_name" {
  type = string
}

variable "enable_default_user" {
  type = bool
}

variable "password" {
  type      = string
  sensitive = true
  default   = null
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
  name              = var.subscription_name
  plan_id           = data.rediscloud_essentials_plan.example.id
  payment_method_id = data.rediscloud_payment_method.card.id
}

resource "rediscloud_essentials_database" "example" {
  subscription_id     = rediscloud_essentials_subscription.example.id
  name                = var.database_name
  enable_default_user = var.enable_default_user
  password            = var.password
  data_persistence    = "none"
  replication         = false

  tags = {
    "envaaaa" = "qaaaa"
  }
}
