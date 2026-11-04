resource "rediscloud_agent_memory" "example" {
  name        = "store"
  database_id = 123

  long_term_memory_exclusions {
    enabled = true

    built_in_detectors {
      enabled = true

      detectors {
        id      = "email"
        enabled = true
        action  = ""
      }
    }
  }
}
