variable "value_input" {
  type    = string
  default = "FAKE-FUTURE-STRATEGY"
}

resource "terraform_data" "value" { input = var.value_input }
resource "rediscloud_agent_memory" "example" {
  name                = "store"
  database_id         = 123
  extraction_strategy = terraform_data.value.output

  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
