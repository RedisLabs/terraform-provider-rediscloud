variable "name" {
  type = string
}

data "rediscloud_essentials_plan" "by_name" {
  name = var.name
}