locals {
  name = "__NAME__"
}

data "rediscloud_acl_rule" "test" {
  name = local.name
}
