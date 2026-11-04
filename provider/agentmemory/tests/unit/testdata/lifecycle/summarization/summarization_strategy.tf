variable "settings" {
  type = object({ enabled = bool, strategy = optional(string), source = optional(string) })
}

resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  summarization {
    enabled          = var.settings.enabled
    trigger_strategy = var.settings.strategy
  }
}
