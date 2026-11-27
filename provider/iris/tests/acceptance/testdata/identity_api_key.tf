variable "expires_at" { default = null }
variable "name" { type = string }
variable "key_name" { type = string }
variable "tenant" { default = "" }
variable "extra_grant" { default = false }

resource "rediscloud_iris_agent_memory" "test" {
  name                     = var.name
  database_id              = local.database_id
  short_memory_ttl_seconds = 86400
}
resource "rediscloud_iris_agent_memory" "other" {
  name                     = "${var.name}-other"
  database_id              = local.database_id
  short_memory_ttl_seconds = 86400
}
resource "rediscloud_iris_identity_api_key" "test" {
  name       = var.key_name
  expires_at = var.expires_at
  tenant     = var.tenant == "" ? null : var.tenant
  dynamic "grant" {
    for_each = var.extra_grant ? toset([rediscloud_iris_agent_memory.test.id, rediscloud_iris_agent_memory.other.id]) : toset([rediscloud_iris_agent_memory.test.id])
    content {
      product       = "memory"
      resource_type = "mem-store"
      resource_id   = grant.value
      actions       = ["read"]
    }
  }
}
