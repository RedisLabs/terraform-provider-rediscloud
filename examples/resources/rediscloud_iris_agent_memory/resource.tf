resource "rediscloud_iris_agent_memory" "example" {
  name = "application-memory"
  database_json = jsonencode({
    databaseId = "your-database-id"
  })
  long_term_memory_json = jsonencode({
    ttlSeconds = 86400
  })
  flush_on_delete = false
}
