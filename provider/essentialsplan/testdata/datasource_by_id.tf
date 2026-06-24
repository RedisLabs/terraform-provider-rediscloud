variable "id" {
  type = number
}

data "rediscloud_essentials_plan" "by_id" {
  id = var.id
}