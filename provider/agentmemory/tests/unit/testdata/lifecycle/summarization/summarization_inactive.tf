resource "rediscloud_agent_memory" "example" {
  name        = "fixture"
  database_id = 123
  summarization {
    enabled          = false
    trigger_strategy = "event_count"
    event_count {
      threshold    = 20
      retain_count = 10
    }
  }
}
