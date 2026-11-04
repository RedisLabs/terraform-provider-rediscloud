variable "values" {
  type = object({ threshold = number, retain = number })
}

resource "terraform_data" "values" { input = var.values }
resource "rediscloud_agent_memory" "example" {
  name                       = "event-count-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  summarization {
    enabled = true
    event_count {
      threshold    = terraform_data.values.output.threshold
      retain_count = terraform_data.values.output.retain
    }
  }
}
