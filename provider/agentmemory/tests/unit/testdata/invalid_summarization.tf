resource "rediscloud_agent_memory" "example" {
  name        = "store"
  database_id = 123

  summarization {
    enabled          = true
    trigger_strategy = "event_count"

    event_count {
      threshold    = 10
      retain_count = 10
    }
  }
}
