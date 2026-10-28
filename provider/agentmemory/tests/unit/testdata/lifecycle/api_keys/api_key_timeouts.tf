variable "create" {
  type    = string
  default = "2m"
}

variable "read" {
  type    = string
  default = "3m"
}

variable "delete" {
  type    = string
  default = "4m"
}

resource "rediscloud_agent_memory_api_key" "runtime" {
  store_id = "store-1"
  name     = "runtime"
  timeouts {
    create = var.create
    read   = var.read
    delete = var.delete
  }
}
