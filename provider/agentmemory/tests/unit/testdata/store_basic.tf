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

resource "rediscloud_agent_memory" "example" {
  name                       = var.name
  database_id                = var.database_id
  short_term_ttl_seconds     = var.short_ttl
  long_term_ttl_seconds      = var.long_ttl
  extraction_cadence_seconds = var.cadence
}
