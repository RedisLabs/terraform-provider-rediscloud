resource "rediscloud_iris_agent_memory_type" "example" {
  store_id           = rediscloud_iris_agent_memory.example.id
  name               = "preferences"
  description        = "User preferences"
  extraction_prompt  = "Extract the user's preferences."
  extraction_enabled = true
  field {
    name        = "topic"
    description = "Preference topic"
    type        = "STRING"
  }
}
