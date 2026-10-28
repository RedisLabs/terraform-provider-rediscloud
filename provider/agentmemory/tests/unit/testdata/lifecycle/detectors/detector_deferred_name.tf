variable "name" {
  type = string
}

resource "terraform_data" "name" { input = var.name }
resource "rediscloud_agent_memory" "example" {
  name                       = "reserved-detector-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  long_term_memory_exclusions {
    enabled = true

    custom_detectors {
      enabled = true
      detectors {
        name    = tostring(terraform_data.name.output)
        enabled = true
        action  = "redact"
        matcher {
          kind = "regex"
          regex { pattern = "ticket-[0-9]+" }
        }
      }
    }
  }
}
