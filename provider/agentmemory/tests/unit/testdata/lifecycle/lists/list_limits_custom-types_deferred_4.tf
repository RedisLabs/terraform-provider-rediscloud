# Independent dependencies keep existing names known when a type is added.
resource "terraform_data" "type_name_0" { input = "custom_0" }
resource "terraform_data" "type_name_1" { input = "custom_1" }
resource "terraform_data" "type_name_2" { input = "custom_2" }
resource "terraform_data" "type_name_3" { input = "custom_3" }
resource "rediscloud_agent_memory" "example" {
  name                       = "list-limits-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  custom_memory_types {
    name        = terraform_data.type_name_0.output
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = "str"
    }
  }
  custom_memory_types {
    name        = terraform_data.type_name_1.output
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = "str"
    }
  }
  custom_memory_types {
    name        = terraform_data.type_name_2.output
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = "str"
    }
  }
  custom_memory_types {
    name        = terraform_data.type_name_3.output
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = "str"
    }
  }
}
