variable "value_input" {
  type    = string
  default = "User preferences"
}

resource "terraform_data" "value" { input = var.value_input }
resource "rediscloud_agent_memory" "example" {
  name                       = "custom-type-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  custom_memory_types {
    name        = "preference"
    description = terraform_data.value.output
    fields {
      name        = "topic"
      type        = "str"
      description = "Topic"
    }
    fields {
      name        = "confidence"
      type        = "float"
      description = "Confidence"
    }
    extraction_strategy {
      prompt  = "Extract preferences"
      enabled = true
    }
  }


}
