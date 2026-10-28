variable "value_input" {
  type    = number
  default = 456
}

resource "terraform_data" "value" { input = var.value_input }
resource "rediscloud_agent_memory" "example" {
  name                = "store"
  database_id         = terraform_data.value.output
  extraction_strategy = "INSTRUCT"

  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
