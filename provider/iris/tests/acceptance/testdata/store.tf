variable "name" { type = string }
variable "short_ttl" { default = 86400 }
variable "field_type" { default = "STRING" }
variable "database_id" { default = "51785871" }
resource "rediscloud_iris_agent_memory" "test" {
  name                    = var.name
  database_json           = jsonencode({ databaseId = var.database_id })
  short_memory_json       = jsonencode({ ttlSeconds = var.short_ttl })
  long_term_memory_json   = var.models_json != null ? var.models_json : jsonencode({ ttlSeconds = 31536000 })
  extraction_cadence_json = jsonencode({ activeIntervalSeconds = 300 })
  summarization_json      = jsonencode({ enabled = true, triggerStrategy = "EVENT_COUNT", eventCount = { threshold = 20, retainCount = 10 } })
  custom_memory_types_json = jsonencode([{
    name               = "preference", description = "Durable preferences for acceptance testing",
    fields             = [{ name = "topic", description = "Preference topic", type = var.field_type }],
    extractionStrategy = { enabled = true, prompt = "Extract durable preferences explicitly stated by the user." }
  }])
}
variable "models_json" {
  type      = string
  default   = null
  sensitive = true
}
