variable "lookup_name" {
  type    = string
  default = null
}

data "rediscloud_essentials_database" "by_name" {
  subscription_id = rediscloud_essentials_subscription.example.id
  name            = coalesce(var.lookup_name, rediscloud_essentials_database.example.name)
}

data "rediscloud_essentials_database" "by_id" {
  subscription_id = rediscloud_essentials_subscription.example.id
  db_id           = rediscloud_essentials_database.example.db_id
}
