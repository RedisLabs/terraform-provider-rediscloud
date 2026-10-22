variable "lookup_name" {
  type    = string
  default = null
}

variable "lookup_id" {
  type    = number
  default = null
}

data "rediscloud_essentials_subscription" "by_name" {
  name = coalesce(var.lookup_name, rediscloud_essentials_subscription.example.name)
}

data "rediscloud_essentials_subscription" "by_id" {
  id = coalesce(var.lookup_id, rediscloud_essentials_subscription.example.id)
}
