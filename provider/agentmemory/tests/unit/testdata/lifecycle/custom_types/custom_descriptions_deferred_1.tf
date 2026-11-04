variable "values" {
  type = list(string)
}

resource "terraform_data" "description_0" { input = var.values[0] }
resource "rediscloud_agent_memory" "example" {
  name                       = "list-limits-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  custom_memory_types {
    name        = "custom_0"
    description = terraform_data.description_0.output
    fields {
      name        = "topic"
      description = terraform_data.description_0.output
      type        = "str"
    }
  }
}
