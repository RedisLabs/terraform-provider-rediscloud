variable "entry_count" {
  type = number
}

resource "terraform_data" "names" { input = [for i in range(var.entry_count) : "custom_${i}"] }
resource "rediscloud_agent_memory" "example" {
  name                       = "list-limits-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  long_term_memory_exclusions {
    enabled = true
    custom_detectors {
      enabled = true
      dynamic "detectors" {
        for_each = range(var.entry_count)
        content {

          name    = terraform_data.names.output[detectors.value]
          enabled = true
          action  = "redact"
          matcher {
            kind = "regex"
            regex { pattern = "secret-[0-9]+" }
          }

        }
      }

    }
  }
}
