resource "rediscloud_iris_agent_memory_type" "example" {
  store_id    = rediscloud_iris_agent_memory.example.id
  name        = "preferences"
  description = "User preferences"
  fields_json = jsonencode([{
    name        = "topic"
    description = "Preference topic"
    type        = "STRING"
  }])
  extraction_strategy_json = jsonencode({
    prompt  = "Extract the user's preferences."
    enabled = true
  })
}
