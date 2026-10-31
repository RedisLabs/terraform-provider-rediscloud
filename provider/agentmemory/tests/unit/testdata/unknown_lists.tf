variable "kind" {
  type = string
}

variable "items" {
  type = list(string)
}

resource "terraform_data" "items" {
  input = var.items
}

resource "rediscloud_agent_memory" "example" {
  name                       = "dynamic-list-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300

  dynamic "custom_memory_types" {
    for_each = var.kind == "memory_types" ? terraform_data.items.output : var.kind == "fields" ? ["preference"] : []
    content {
      name        = custom_memory_types.value
      description = "Durable preferences"
      dynamic "fields" {
        for_each = var.kind == "fields" ? terraform_data.items.output : ["topic"]
        content {
          name        = fields.value
          description = "Preference topic"
          type        = "str"
        }
      }
    }
  }

  dynamic "long_term_memory_exclusions" {
    for_each = contains(["built_in", "custom"], var.kind) ? [true] : []
    content {
      enabled = true
      dynamic "built_in_detectors" {
        for_each = var.kind == "built_in" ? [true] : []
        content {
          enabled = true
          dynamic "detectors" {
            for_each = terraform_data.items.output
            content {
              id      = detectors.value
              enabled = true
              action  = "redact"
            }
          }
        }
      }
      dynamic "custom_detectors" {
        for_each = var.kind == "custom" ? [true] : []
        content {
          enabled = true
          dynamic "detectors" {
            for_each = terraform_data.items.output
            content {
              name    = detectors.value
              enabled = true
              action  = "redact"
              matcher {
                kind = "regex"
                regex {
                  pattern = "ticket-[0-9]+"
                }
              }
            }
          }
        }
      }
    }
  }
}
