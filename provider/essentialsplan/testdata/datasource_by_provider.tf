variable "id" {
  type = number
}

variable "provider_name" {
  type = string
}

data "rediscloud_essentials_plan" "by_provider" {
  id             = var.id
  cloud_provider = var.provider_name
}