variable "name" {
  type = string
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
      threshold    = 21
      retain_count = 11
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
      enabled = true
      prompt  = "test"
    }
  }

  custom_memory_types {
    name        = "support_case"
    description = "Persistent support case details"

    fields {
      name        = "case_priority"
      type        = "int"
      description = "case priority"
    }

    extraction_strategy {
      enabled = true
      prompt  = "Extract durable support case details."
    }
  }

  long_term_memory_exclusions {
    enabled = true

    semantic {
      enabled = true
      prompt  = "test"
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
        action  = "redact"
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
