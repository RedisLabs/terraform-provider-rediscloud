variable "value" { type = string }
variable "mutable_value" { type = string }
variable "deferred" { type = bool }

resource "terraform_data" "value" { input = var.value }
resource "terraform_data" "mutable" { input = var.mutable_value }

locals {
  value   = var.deferred ? terraform_data.value.output : var.value
  mutable = var.deferred ? terraform_data.mutable.output : var.mutable_value
}

resource "rediscloud_agent_memory" "example" {
  name                       = "empty-strings-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300

  summarization {
    enabled          = true
    trigger_strategy = local.mutable
    event_count {
      threshold    = 10
      retain_count = 2
    }
  }
  long_term_memory_exclusions {
    enabled = true
    built_in_detectors {
      enabled = true
      detectors {
        id      = "email"
        enabled = true
        action  = local.mutable
      }
    }
    custom_detectors {
      enabled = true
      detectors {
        name    = local.value
        enabled = true
        action  = local.mutable
        matcher {
          kind = local.mutable
          regex { pattern = local.mutable }
        }
      }
    }
  }
  custom_memory_types {
    name        = local.value
    description = local.value
    fields {
      name        = local.value
      description = local.value
      type        = local.value
    }
    extraction_strategy {
      prompt  = local.mutable
      enabled = true
    }
  }
}
