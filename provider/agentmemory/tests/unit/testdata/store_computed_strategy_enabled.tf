variable "name" {
  type = string
}

variable "operation_timeout" {
  type    = string
  default = null
}

resource "rediscloud_agent_memory" "example" {
  dynamic "timeouts" {
    for_each = var.operation_timeout == null ? [] : [var.operation_timeout]
    content {
      create = timeouts.value
      read   = timeouts.value
      update = timeouts.value
      delete = timeouts.value
    }
  }

  name                       = var.name
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 604800
  extraction_cadence_seconds = 300

  custom_memory_types {
    name        = "preference"
    description = "User preferences"

    extraction_strategy {
      prompt = "Extract preferences"
    }
  }
}
