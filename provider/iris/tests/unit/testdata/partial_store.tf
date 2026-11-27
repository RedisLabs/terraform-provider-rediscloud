variable "memory_url" { type = string }
variable "database_id" { default = "db-1" }
variable "partial" { default = false }
variable "edit" { default = "summary_threshold" }

provider "rediscloud" {
  iris_agent_memory_url   = var.memory_url
  iris_agent_memory_token = "memory-token"
}

resource "rediscloud_iris_agent_memory" "store" {
  name                           = "partial-settings"
  database_id                    = var.database_id
  summarization_enabled          = var.partial ? null : true
  summarization_trigger_strategy = var.partial ? (var.edit == "summary_trigger" ? "EVENT_COUNT" : null) : "EVENT_COUNT"
  summarization_event_threshold  = var.partial ? (var.edit == "summary_threshold" ? 200 : null) : 100
  summarization_retain_count     = var.partial ? (var.edit == "summary_retain" ? 0 : null) : 10
  exclusions_enabled             = var.partial ? null : true
  semantic_exclusions_enabled    = var.partial ? (var.edit == "semantic_enabled" ? false : null) : true
  semantic_exclusions_prompt     = var.partial ? (var.edit == "semantic_prompt" ? "new prompt" : null) : "private data"
  built_in_detectors_enabled     = var.partial ? (var.edit == "built_in_enabled" ? false : null) : true
  custom_detectors_enabled       = var.partial ? (var.edit == "custom_enabled" ? false : null) : true

  built_in_detector {
    id      = "email"
    enabled = true
    action  = var.partial && var.edit == "detector_action" ? "REDACT" : "DROP"
  }
  custom_detector {
    name    = "private"
    enabled = true
    pattern = "secret[0-9]+"
  }
}
