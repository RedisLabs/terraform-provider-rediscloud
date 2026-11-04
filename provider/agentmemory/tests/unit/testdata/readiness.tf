variable "name" {
  type    = string
  default = "store"
}
resource "test_agent_memory" "example" {
  name                       = var.name
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  timeouts {
    create = "5s"
    update = "5s"
  }
}
resource "test_agent_memory_api_key" "dependent" {
  store_id = test_agent_memory.example.id
  name     = "runtime"
}
