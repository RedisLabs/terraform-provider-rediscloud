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
variable "customer_models" {
  type    = bool
  default = false
}
variable "llm_model" {
  type    = string
  default = "gpt-4o-mini"
}
variable "embedding_model" {
  type    = string
  default = "text-embedding-3-small"
}
variable "key_version" {
  type    = number
  default = 1
}
variable "llm_api_key" {
  type      = string
  sensitive = true
  ephemeral = true
  default   = null
}
variable "embedding_api_key" {
  type      = string
  sensitive = true
  ephemeral = true
  default   = null
}

resource "rediscloud_agent_memory" "test" {
  name                       = "${var.name}-${var.store_suffix}"
  database_id                = rediscloud_subscription_database.test[var.database_index].db_id
  short_term_ttl_seconds     = var.short_ttl
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300

  dynamic "llm" {
    for_each = var.customer_models ? [true] : []
    content {
      provider = "openai"
      model    = var.llm_model
      credentials {
        type            = "apiKey"
        api_key         = var.llm_api_key
        api_key_version = var.key_version
      }
    }
  }

  dynamic "embedding" {
    for_each = var.customer_models ? [true] : []
    content {
      provider = "openai"
      model    = var.embedding_model
      credentials {
        type            = "apiKey"
        api_key         = var.embedding_api_key
        api_key_version = var.key_version
      }
    }
  }

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
