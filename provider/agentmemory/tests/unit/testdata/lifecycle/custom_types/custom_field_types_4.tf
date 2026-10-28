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
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = var.values[0]
    }
  }
  custom_memory_types {
    name        = "custom_1"
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = var.values[1]
    }
  }
  custom_memory_types {
    name        = "custom_2"
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = var.values[2]
    }
  }
  custom_memory_types {
    name        = "custom_3"
    description = "Custom memory"
    fields {
      name        = "topic"
      description = "Topic"
      type        = var.values[3]
    }
  }
}
