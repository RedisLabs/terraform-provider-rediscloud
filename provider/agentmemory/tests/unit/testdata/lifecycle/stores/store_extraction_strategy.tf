variable "name" {
  type = string
}

variable "strategy" {
  type = string
}

resource "rediscloud_agent_memory" "example" {
  name                = var.name
  database_id         = 123
  extraction_strategy = var.strategy
}
