resource "rediscloud_iris_agent_memory" "example" {
  name                         = "application-memory"
  database_id                  = "your-database-id"
  long_term_memory_ttl_seconds = 86400
  flush_on_delete              = false
}
