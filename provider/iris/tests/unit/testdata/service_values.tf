variable "memory_url" { type = string }
variable "identity_url" { type = string }
variable "name" { type = string }
provider "rediscloud" {
  iris_agent_memory_url   = var.memory_url
  iris_agent_memory_token = "memory-token"
  iris_identity_url       = var.identity_url
  iris_identity_token     = "identity-token"
}
resource "rediscloud_iris_agent_memory" "store" {
  name                         = var.name
  database_id                  = "future-database-id"
  extraction_strategy          = "FUTURE_STRATEGY"
  ownership_mode               = "FUTURE_POLICY"
  long_term_memory_ttl_seconds = 31536001
  extraction_interval_seconds  = 601
  custom_memory_type {
    name        = var.name
    description = var.name
    field {
      name        = var.name
      description = var.name
      type        = "FUTURE_FIELD_TYPE"
    }
  }
  exclusions_enabled       = true
  custom_detectors_enabled = true
  dynamic "custom_detector" {
    for_each = range(40)
    content {
      name    = "detector_${custom_detector.key}"
      enabled = true
      action  = "FUTURE_ACTION"
      pattern = "pattern[0-9]+"
    }
  }
}
resource "rediscloud_iris_agent_memory_type" "type" {
  store_id    = "a-future:opaque-store-identifier"
  name        = var.name
  description = var.name
}
resource "rediscloud_iris_identity_api_key" "key" {
  name    = var.name
  user_id = var.name
  grant {
    product       = "future-product"
    resource_type = "future-resource"
    resource_id   = "future-id"
    actions       = ["future-action"]
  }
}
