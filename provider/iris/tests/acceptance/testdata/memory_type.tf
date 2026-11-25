variable "name" { type = string }
variable "create_type" { default = true }
variable "description" { default = "Durable acceptance-test preferences" }
variable "field_type" { default = "STRING" }
variable "prompt" { default = "Extract durable preferences explicitly stated by the user." }

resource "rediscloud_iris_agent_memory" "test" {
  name                     = var.name
  database_id              = local.database_id
  short_memory_ttl_seconds = 86400
}

resource "rediscloud_iris_agent_memory_type" "test" {
  count              = var.create_type ? 1 : 0
  store_id           = rediscloud_iris_agent_memory.test.id
  name               = "acceptance_preference"
  description        = var.description
  extraction_prompt  = var.prompt
  extraction_enabled = true

  field {
    name        = "topic"
    description = "Preference topic"
    type        = var.field_type
  }
}

resource "rediscloud_iris_agent_memory_type" "sibling" {
  count              = var.create_type ? 2 : 0
  store_id           = rediscloud_iris_agent_memory.test.id
  name               = "acceptance_sibling_${count.index}"
  description        = "Sibling type for concurrent registry writes"
  extraction_prompt  = var.prompt
  extraction_enabled = true
  field {
    name        = "topic"
    description = "Preference topic"
    type        = "STRING"
  }
}
