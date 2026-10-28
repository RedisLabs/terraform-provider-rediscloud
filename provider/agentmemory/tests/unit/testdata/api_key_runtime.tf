variable "store_id" {
  type    = string
  default = "store-1"
}

variable "name" {
  type    = string
  default = "runtime"
}

resource "rediscloud_agent_memory_api_key" "runtime" {
  store_id = var.store_id
  name     = var.name
}
