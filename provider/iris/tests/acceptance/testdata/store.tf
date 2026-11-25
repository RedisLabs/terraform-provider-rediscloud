variable "name" { type = string }
variable "store_name" { default = null }
variable "customer_models" { default = false }
variable "short_ttl" { default = 86400 }
variable "field_type" { default = "STRING" }
variable "llm_api_key" {
  type      = string
  default   = null
  sensitive = true
  ephemeral = true
}
variable "embedding_api_key" {
  type      = string
  default   = null
  sensitive = true
  ephemeral = true
}
resource "rediscloud_iris_agent_memory" "test" {
  name                           = coalesce(var.store_name, var.name)
  database_id                    = local.database_id
  short_memory_ttl_seconds       = var.short_ttl
  long_term_memory_ttl_seconds   = 31536000
  extraction_interval_seconds    = 300
  summarization_enabled          = true
  summarization_trigger_strategy = "EVENT_COUNT"
  summarization_event_threshold  = 20
  summarization_retain_count     = 10
  llm_provider                   = var.customer_models ? "openai" : null
  llm_model                      = var.customer_models ? "gpt-4o-mini" : null
  llm_api_key                    = var.llm_api_key
  llm_api_key_version            = var.customer_models ? 1 : null
  embedding_provider             = var.customer_models ? "openai" : null
  embedding_model                = var.customer_models ? "text-embedding-3-small" : null
  embedding_api_key              = var.embedding_api_key
  embedding_api_key_version      = var.customer_models ? 1 : null
  custom_memory_type {
    name               = "preference"
    description        = "Durable preferences for acceptance testing"
    extraction_prompt  = "Extract durable preferences explicitly stated by the user."
    extraction_enabled = true
    field {
      name        = "topic"
      description = "Preference topic"
      type        = var.field_type
    }
  }
}
