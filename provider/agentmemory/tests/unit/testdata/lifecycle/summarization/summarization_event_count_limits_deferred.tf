variable "threshold" {
  type = number
}

variable "retain" {
  type = number
}

resource "terraform_data" "counts" { input = { threshold = var.threshold, retain = var.retain } }
resource "rediscloud_agent_memory" "example" {
  name                       = "event-count-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  summarization {
    enabled = true
    event_count {
      threshold    = terraform_data.counts.output.threshold
      retain_count = terraform_data.counts.output.retain
    }
  }
}
