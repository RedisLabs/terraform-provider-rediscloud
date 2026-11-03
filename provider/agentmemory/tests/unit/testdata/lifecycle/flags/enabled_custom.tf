variable "enabled" {
  type = bool
}

resource "terraform_data" "flag" { input = var.enabled }
resource "rediscloud_agent_memory" "example" {
  name                       = "unknown-flags-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  long_term_memory_exclusions {
    enabled = true
    custom_detectors {
      enabled = terraform_data.flag.output
    }
  }
}
