terraform {
  required_providers {
    rediscloud = {
      source = "RedisLabs/rediscloud"
    }
  }
}

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
