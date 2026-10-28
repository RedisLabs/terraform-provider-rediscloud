resource "rediscloud_agent_memory" "example" {
  name        = "store"
  database_id = 123

  long_term_memory_exclusions {
    enabled = true

    custom_detectors {
      enabled = true

      detectors {
        name    = "tenant_id"
        enabled = true
        action  = "redact"
      }
    }
  }
}
