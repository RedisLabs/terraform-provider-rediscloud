variable "name" {
  type = string
}

resource "rediscloud_agent_memory" "example" {
  name                       = var.name
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300

  summarization {
    enabled = false
  }

  custom_memory_types {
    name        = "custom_name"
    description = "test"

    fields {
      name        = "field1"
      type        = "str"
      description = "capture the field"
    }

    extraction_strategy {
      enabled = true
      prompt  = "test"
    }
  }

  long_term_memory_exclusions {
    enabled = true

    semantic {
      enabled = false
    }

    built_in_detectors {
      enabled = false
    }

    custom_detectors {
      enabled = false
    }
  }
}
