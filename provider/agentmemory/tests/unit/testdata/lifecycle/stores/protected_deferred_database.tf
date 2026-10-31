resource "terraform_data" "value" { input = 123 }
resource "rediscloud_agent_memory" "example" {
  name                = "store"
  database_id         = terraform_data.value.output
  extraction_strategy = "INSTRUCT"
  lifecycle { prevent_destroy = true }
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
}
