variable "database_id" {
  type    = number
  default = 456
}

variable "extraction_strategy" {
  type    = string
  default = "INSTRUCT"
}

resource "rediscloud_agent_memory" "example" {
  name                = "store"
  database_id         = var.database_id
  extraction_strategy = var.extraction_strategy
  lifecycle { prevent_destroy = true }
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
