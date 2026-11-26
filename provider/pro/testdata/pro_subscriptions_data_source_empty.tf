variable "filter_name" {
  type = string
}

# No resources: exercises the empty-result path. filter_name is a random name that
# matches no subscription, so the data source returns an empty list rather than erroring.
data "rediscloud_subscriptions" "example" {
  filters {
    name = var.filter_name
  }
}
