data "rediscloud_subscription" "example" {
  name       = rediscloud_subscription.example.name
  depends_on = [rediscloud_subscription_database.example]
}
