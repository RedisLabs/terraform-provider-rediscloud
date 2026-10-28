resource "rediscloud_agent_memory" "example" {
  name                       = "store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300

  custom_memory_types {
    name        = "duplicate_name"
    description = "first type"

    fields {
      name        = "first_field"
      type        = "str"
      description = "first field"
    }

    extraction_strategy {
      enabled = true
      prompt  = "Extract the first type."
    }
  }

  custom_memory_types {
    name        = "duplicate_name"
    description = "second type"

    fields {
      name        = "second_field"
      type        = "str"
      description = "second field"
    }

    extraction_strategy {
      enabled = true
      prompt  = "Extract the second type."
    }
  }
}
