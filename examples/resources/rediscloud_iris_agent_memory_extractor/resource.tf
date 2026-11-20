resource "rediscloud_iris_agent_memory_extractor" "example" {
  store_id    = rediscloud_iris_agent_memory.example.id
  name        = "document-profile"
  memory_type = rediscloud_iris_agent_memory_type.example.name
  source      = "DOCUMENTS"
  strategy    = "CUSTOM"
  prompt      = "Extract the user's preferences."
  enabled     = true
}
