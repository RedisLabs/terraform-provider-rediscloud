variable "provider_name" {
  type = string
}

data "rediscloud_regions" "example" {
  provider_name = var.provider_name
}
