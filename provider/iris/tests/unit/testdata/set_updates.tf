variable "memory_url" { type = string }
variable "identity_url" { type = string }
variable "kind" { type = string }
variable "items" { type = set(string) }
variable "detector_action" { default = "DROP" }
provider "rediscloud" {
  iris_agent_memory_url   = var.memory_url
  iris_agent_memory_token = "memory-token"
  iris_identity_url       = var.identity_url
  iris_identity_token     = "identity-token"
}
resource "rediscloud_iris_identity_api_key" "key" {
  count      = var.kind == "key" ? 1 : 0
  name       = "set-key"
  expires_at = 0
  dynamic "grant" {
    for_each = var.items
    content {
      product       = "memory"
      resource_type = "mem-store"
      resource_id   = grant.value
      actions       = ["read"]
    }
  }
}
resource "rediscloud_iris_agent_memory" "store" {
  count                      = var.kind == "store" ? 1 : 0
  name                       = "set-store"
  database_id                = "db-1"
  exclusions_enabled         = true
  built_in_detectors_enabled = true
  custom_detectors_enabled   = true
  dynamic "built_in_detector" {
    for_each = var.items
    content {
      id      = built_in_detector.value
      enabled = true
      action  = built_in_detector.value == "first" && var.detector_action != "" ? var.detector_action : null
    }
  }
  dynamic "custom_detector" {
    for_each = var.items
    content {
      name    = custom_detector.value
      enabled = true
      action  = custom_detector.value == "first" && var.detector_action != "" ? var.detector_action : null
      pattern = "secret[0-9]+"
    }
  }
}
