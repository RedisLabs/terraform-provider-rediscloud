variable "entry_count" {
  type = number
}

locals { ids = ["credit-card", "email", "ip-address", "phone", "us-ssn"] }
resource "rediscloud_agent_memory" "example" {
  name                       = "list-limits-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  long_term_memory_exclusions {
    enabled = true
    built_in_detectors {
      enabled = true
      dynamic "detectors" {
        for_each = range(var.entry_count)
        content {

          id      = local.ids[detectors.value % length(local.ids)]
          enabled = true
          action  = "redact"

        }
      }

    }
  }
}
