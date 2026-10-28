variable "values" {
  type = list(string)
}

resource "terraform_data" "field_type_3" { input = var.values[3] }
resource "terraform_data" "field_type_2" { input = var.values[2] }
resource "terraform_data" "field_type_1" { input = var.values[1] }
resource "terraform_data" "field_type_0" { input = var.values[0] }
resource "rediscloud_agent_memory" "example" {
  name                       = "list-limits-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  custom_memory_types {
    name        = "custom_0"
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = terraform_data.field_type_0.output
    }
  }
  custom_memory_types {
    name        = "custom_1"
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = terraform_data.field_type_1.output
    }
  }
  custom_memory_types {
    name        = "custom_2"
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = terraform_data.field_type_2.output
    }
  }
  custom_memory_types {
    name        = "custom_3"
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = terraform_data.field_type_3.output
    }
  }
}
