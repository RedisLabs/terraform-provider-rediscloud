variable "input" {
  type = string
}

resource "terraform_data" "source" { input = var.input }
resource "rediscloud_agent_memory" "audit" {
  name                       = "audit-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  custom_memory_types {
    name        = split(":", tostring(terraform_data.source.output))[0]
    description = "Shopping preferences"
    fields {
      name        = "topic"
      description = "Preference topic"
      type        = "str"
    }
    extraction_strategy {
      prompt  = "Extract preferences"
      enabled = true
    }
  }
  custom_memory_types {
    name        = "support"
    description = "Support requests"
    extraction_strategy {
      prompt  = "Extract support requests"
      enabled = true
    }
  }
}
