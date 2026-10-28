variable "name" {
  type = string
}

variable "threshold" {
  type    = number
  default = 21
}

variable "retain_count" {
  type    = number
  default = 11
}

variable "custom_prompt" {
  type    = string
  default = "test"
}

variable "custom_enabled" {
  type    = bool
  default = true
}

variable "semantic_prompt" {
  type    = string
  default = "test"
}

variable "email_action" {
  type    = string
  default = "redact"
}

resource "rediscloud_agent_memory" "example" {
  name                       = var.name
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300

  summarization {
    enabled          = true
    trigger_strategy = "event_count"

    event_count {
      threshold    = var.threshold
      retain_count = var.retain_count
    }
  }

  custom_memory_types {
    name        = "custom_name"
    description = "test"

    fields {
      name        = "field1"
      type        = "str"
      description = "capture the field"
    }

    extraction_strategy {
      enabled = var.custom_enabled
      prompt  = var.custom_prompt
    }
  }

  long_term_memory_exclusions {
    enabled = true

    semantic {
      enabled = true
      prompt  = var.semantic_prompt
    }

    built_in_detectors {
      enabled = true

      detectors {
        id      = "credit-card"
        enabled = true
        action  = "redact"
      }

      detectors {
        id      = "email"
        enabled = true
        action  = var.email_action
      }
    }

    custom_detectors {
      enabled = true

      detectors {
        name    = "detector"
        enabled = true
        action  = "redact"

        matcher {
          kind = "regex"

          regex {
            pattern = "ACCT-[9]"
          }
        }
      }
    }
  }
}
