variable "name" {
  type = string
}

data "rediscloud_acl_rule" "test" {
  name = var.name
}
