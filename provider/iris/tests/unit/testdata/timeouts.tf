variable "memory_url" { type = string }
variable "identity_url" { type = string }
variable "read_timeout" { default = "5m" }
variable "expires_at" { default = 0 }
provider "rediscloud" {
  iris_agent_memory_url   = var.memory_url
  iris_agent_memory_token = "memory-token"
  iris_identity_url       = var.identity_url
  iris_identity_token     = "identity-token"
}
resource "rediscloud_iris_agent_memory" "store" {
  name        = "timeout-store"
  database_id = "db-1"
  timeouts {
    create = "10m"
    read   = var.read_timeout
    update = "10m"
    delete = "10m"
  }
}
resource "rediscloud_iris_agent_memory_type" "type" {
  store_id    = rediscloud_iris_agent_memory.store.id
  name        = "profile"
  description = "Profile"
  timeouts {
    create = "10m"
    read   = var.read_timeout
    update = "10m"
    delete = "10m"
  }
}
resource "rediscloud_iris_identity_api_key" "key" {
  name       = "timeout-key"
  expires_at = var.expires_at
  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = rediscloud_iris_agent_memory.store.id
    actions       = ["read"]
  }
  timeouts {
    create = "10m"
    read   = var.read_timeout
    update = "10m"
    delete = "10m"
  }
}
