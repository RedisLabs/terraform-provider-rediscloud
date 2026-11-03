variable "action" {
  type = string
}

resource "rediscloud_agent_memory" "example" {
  name                       = "reserved-detector-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  long_term_memory_exclusions {
    enabled = true
    built_in_detectors {
      enabled = true
      detectors {
        id      = "email"
        enabled = true
        action  = var.action
      }
    }

    custom_detectors {
      enabled = true
      detectors {
        name    = "tenant_id"
        enabled = true
        action  = var.action
        matcher {
          kind = "regex"
          regex { pattern = "ticket-[0-9]+" }
        }
      }
    }
  }
}
