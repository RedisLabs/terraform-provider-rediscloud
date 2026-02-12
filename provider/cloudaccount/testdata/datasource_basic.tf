locals {
  name         = "__NAME__"
}

data "rediscloud_cloud_account" "test" {
exclude_internal_account = true
provider_type = "AWS"
name = local.name
}
