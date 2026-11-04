variable "name" {
  type = string
}

variable "database_id" {
  type    = number
  default = 123
}

variable "short_ttl" {
  type    = number
  default = null
}

variable "long_ttl" {
  type    = number
  default = null
}

variable "cadence" {
  type    = number
  default = null
}

variable "key_name" {
  type    = string
  default = "runtime"
}

resource "rediscloud_agent_memory" "example" {
  name                       = var.name
  database_id                = var.database_id
  short_term_ttl_seconds     = var.short_ttl
  long_term_ttl_seconds      = var.long_ttl
  extraction_cadence_seconds = var.cadence
}
resource "rediscloud_agent_memory_api_key" "runtime" {
  store_id = rediscloud_agent_memory.example.id
  name     = var.key_name
}
