variable "name" {
  type = string
}

resource "terraform_data" "name" { input = var.name }
resource "rediscloud_agent_memory" "example" {
  name                       = "reserved-type-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  custom_memory_types {
    name        = tostring(terraform_data.name.output)
    description = "Custom preferences"
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
}
