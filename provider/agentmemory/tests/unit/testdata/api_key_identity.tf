variable "name" {
  type    = string
  default = "runtime"
}

variable "store_id" {
  type    = string
  default = "store-1"
}

variable "read_timeout" {
  type    = string
  default = "2s"
}

resource "test_agent_memory_api_key" "runtime" {
  name     = var.name
  store_id = var.store_id

  timeouts {
    create = "2s"
    read   = var.read_timeout
    delete = "2s"
  }
}

resource "terraform_data" "key_reference" {
  input = test_agent_memory_api_key.runtime.id
}
