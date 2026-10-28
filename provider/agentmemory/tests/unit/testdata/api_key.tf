variable "store_id" {
  type    = string
  default = "store-1"
}

variable "name" {
  type    = string
  default = "terraform"
}

resource "rediscloud_agent_memory_api_key" "example" {
  store_id = var.store_id
  name     = var.name
}
