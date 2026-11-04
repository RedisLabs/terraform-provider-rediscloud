resource "rediscloud_agent_memory" "example" {
  name        = "summary-defaults"
  database_id = 123
  summarization { enabled = true }
}
