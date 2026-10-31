variable "entry_count" {
  type = number
}

resource "rediscloud_agent_memory" "example" {
  name                       = "list-limits-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  dynamic "custom_memory_types" {
    for_each = range(var.entry_count)
    content {

      name        = "custom_${custom_memory_types.value}"
      description = "Custom memory"
      fields {
        name        = "topic"
        description = "Topic"
        type        = "str"
      }

    }
  }

}
