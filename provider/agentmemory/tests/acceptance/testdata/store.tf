variable "database_index" {
  type    = number
  default = 0
}
variable "store_suffix" {
  type    = string
  default = "memory"
}
variable "key_suffix" {
  type    = string
  default = "runtime"
}
variable "short_ttl" {
  type    = number
  default = 86400
}
variable "field_type" {
  type    = string
  default = "str"
}
resource "rediscloud_agent_memory" "test" {
  name                       = "${var.name}-${var.store_suffix}"
  database_id                = rediscloud_subscription_database.test[var.database_index].db_id
  short_term_ttl_seconds     = var.short_ttl
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300

  summarization {
    enabled          = true
    trigger_strategy = "event_count"
    event_count {
      threshold    = 20
      retain_count = 10
    }
  }

  custom_memory_types {
    name        = "preference"
    description = "Durable preferences for acceptance testing"
    fields {
      name        = "topic"
      description = "Preference topic"
      type        = var.field_type
    }
    extraction_strategy {
      enabled = true
      prompt  = "Extract durable preferences explicitly stated by the user."
    }
  }
}

resource "rediscloud_agent_memory_api_key" "test" {
  store_id = rediscloud_agent_memory.test.id
  name     = "${var.name}-${var.key_suffix}"
}
