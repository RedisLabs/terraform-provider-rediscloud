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
    built_in_detectors {
      enabled = true
      dynamic "detectors" {
        for_each = var.names
        content {
          id      = detectors.value
          enabled = true
          action  = var.action
        }
      }

    }
  }
}
