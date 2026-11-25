variable "memory_url" { type = string }
variable "prompt" { default = "Extract" }
provider "rediscloud" {
  iris_agent_memory_url   = var.memory_url
  iris_agent_memory_token = "memory-token"
}
resource "rediscloud_iris_agent_memory_type" "type" {
  count              = 3
  store_id           = "st_11111111111111111111111111111111"
  name               = "type_${count.index}"
  description        = "Profile"
  extraction_prompt  = var.prompt
  extraction_enabled = true
  field {
    name        = "topic"
    description = "Topic"
    type        = "STRING"
  }
}
