variable "name" {
  type = string
}

data "rediscloud_cloud_account" "test" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = var.name
}
