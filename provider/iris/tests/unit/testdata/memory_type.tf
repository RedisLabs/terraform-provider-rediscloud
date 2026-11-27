variable "memory_url" { type = string }
variable "name" { default = "profile" }
variable "description" { default = "Profile" }
variable "field_type" { default = "STRING" }
variable "with_strategy" { default = true }
variable "enabled_without_prompt" { default = false }
provider "rediscloud" {
  iris_agent_memory_url   = var.memory_url
  iris_agent_memory_token = "memory-token"
}
resource "rediscloud_iris_agent_memory_type" "type" {
  store_id           = "st_11111111111111111111111111111111"
  name               = var.name
  description        = var.description
  extraction_prompt  = var.with_strategy ? "Extract profiles" : null
  extraction_enabled = var.with_strategy ? true : (var.enabled_without_prompt ? false : null)
  field {
    name        = "topic"
    description = "Topic"
    type        = var.field_type
  }
}
