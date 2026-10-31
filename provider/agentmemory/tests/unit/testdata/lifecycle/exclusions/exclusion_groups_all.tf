resource "rediscloud_agent_memory" "example" {
  name        = "remove-groups"
  database_id = 123
  long_term_memory_exclusions {
    enabled = true
    semantic {
      enabled = true
      prompt  = "Exclude secrets"
    }
    built_in_detectors {
      enabled = true
      detectors {
        id      = "email"
        enabled = true
      }
    }
    custom_detectors {
      enabled = true
      detectors {
        name    = "ticket"
        enabled = true
        matcher {
          kind = "regex"
          regex { pattern = "ticket-[0-9]+" }
        }
      }
    }
  }
}
