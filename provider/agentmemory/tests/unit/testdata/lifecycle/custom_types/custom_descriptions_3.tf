variable "values" {
  type = list(string)
}

resource "rediscloud_agent_memory" "example" {
  name                       = "list-limits-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  custom_memory_types {
    name        = "custom_0"
    description = var.values[0]
    fields {
      name        = "topic"
      description = var.values[0]
      type        = "str"
    }
  }
  custom_memory_types {
    name        = "custom_1"
    description = var.values[1]
    fields {
      name        = "topic"
      description = var.values[1]
      type        = "str"
    }
  }
  custom_memory_types {
    name        = "custom_2"
    description = var.values[2]
    fields {
      name        = "topic"
      description = var.values[2]
      type        = "str"
    }
  }
}
