terraform {
  required_providers {
    rediscloud = {
      source  = "RedisLabs/rediscloud"
      version = "2.20.0"
    }
  }
}

provider "rediscloud" {}

variable "subscription_name" {
  type = string
}

variable "database_name" {
  type = string
}

variable "password" {
  type      = string
  sensitive = true
}

variable "tags" {
  type = map(string)
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

resource "rediscloud_essentials_database" "example" {
  subscription_id     = rediscloud_essentials_subscription.example.id
  name                = var.database_name
  enable_default_user = true
  password            = var.password
  data_persistence    = "none"
  replication         = false
  tags                = var.tags
}
