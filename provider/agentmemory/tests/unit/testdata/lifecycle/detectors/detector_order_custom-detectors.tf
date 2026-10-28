variable "names" {
  type = list(string)
}

variable "action" {
  type = string
}

resource "rediscloud_agent_memory" "example" {
  name        = "detector-order-store"
  database_id = 123
  long_term_memory_exclusions {
    enabled = true
    custom_detectors {
      enabled = true
      dynamic "detectors" {
        for_each = var.names
        content {
          name    = detectors.value
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
}
