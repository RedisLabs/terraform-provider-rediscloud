variable "values" {
  type = list(string)
}

resource "rediscloud_agent_memory" "example" {
  name                       = "custom-names-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  long_term_memory_exclusions {
    enabled = true
    custom_detectors {
      enabled = true
      detectors {
        name    = var.values[2]
        enabled = true
        action  = "redact"
        matcher {
          kind = "regex"
          regex { pattern = "secret-[0-9]+" }
        }
      }
    }
  }
  custom_memory_types {
    name        = var.values[0]
    description = "Custom memory"
    fields {
      name        = var.values[0]
      description = "Custom field"
      type        = "str"
    }
  }
  custom_memory_types {
    name        = var.values[1]
    description = "Custom memory"
    fields {
      name        = var.values[1]
      description = "Custom field"
      type        = "str"
    }
  }
  custom_memory_types {
    name        = var.values[2]
    description = "Custom memory"
    fields {
      name        = var.values[2]
      description = "Custom field"
      type        = "str"
    }
  }
}
