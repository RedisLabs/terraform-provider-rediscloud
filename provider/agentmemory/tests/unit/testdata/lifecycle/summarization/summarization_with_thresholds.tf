resource "rediscloud_agent_memory" "example" {
  name        = "summary-defaults"
  database_id = 123
  summarization {
    enabled = true
    event_count {
      threshold    = 12
      retain_count = 6
    }
  }
}
