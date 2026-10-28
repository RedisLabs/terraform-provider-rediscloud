resource "rediscloud_agent_memory" "example" {
  name        = "remove-groups"
  database_id = 123
  long_term_memory_exclusions { enabled = true }
}
