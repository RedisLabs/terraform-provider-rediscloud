resource "test_agent_memory" "example" {
  name        = "store"
  database_id = 123

  timeouts {
    create = "2s"
    delete = "2s"
  }
}
