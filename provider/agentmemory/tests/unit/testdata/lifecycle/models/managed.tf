resource "rediscloud_agent_memory" "example" {
  name                       = "selector-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
