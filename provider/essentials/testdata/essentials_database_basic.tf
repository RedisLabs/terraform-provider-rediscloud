terraform {
  required_providers {
    rediscloud = {
      source = "RedisLabs/rediscloud"
    }
  }
}

variable "subscription_id" {
  type = number
}

variable "database_name" {
  type = string
}

variable "redis_version" {
  type = string
}

variable "password" {
  type      = string
  sensitive = true
}

resource "rediscloud_essentials_database" "example" {
  subscription_id = var.subscription_id
  name            = var.database_name
  protocol        = "redis"
  redis_version   = var.redis_version
  replication     = false
  data_persistence = "none"

  password = var.password

  alert {
    name  = "throughput-higher-than"
    value = 80
  }
}

output "database_id" {
  value = rediscloud_essentials_database.example.db_id
}

output "redis_version" {
  value = rediscloud_essentials_database.example.redis_version
}

output "public_endpoint" {
  value = rediscloud_essentials_database.example.public_endpoint
}
