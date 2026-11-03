variable "values" {
  type = list(string)
}

resource "terraform_data" "name_0" { input = var.values[0] }
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
        name    = terraform_data.name_0.output
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
    name        = terraform_data.name_0.output
    description = "Custom memory"
    fields {
      name        = terraform_data.name_0.output
      description = "Custom field"
      type        = "str"
    }
  }
}
