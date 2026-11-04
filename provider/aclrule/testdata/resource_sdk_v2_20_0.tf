terraform {
  required_providers {
    rediscloud = {
      source  = "RedisLabs/rediscloud"
      version = "2.20.0"
    }
  }
}

provider "rediscloud" {}

variable "name" {
  type = string
}

variable "rule" {
  type = string
}

resource "rediscloud_acl_rule" "test" {
  name = var.name
  rule = var.rule
}
