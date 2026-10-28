variable "prompt" {
  type = string
}

resource "terraform_data" "prompt" { input = var.prompt }
resource "rediscloud_agent_memory" "example" {
  name                       = "semantic-prompt-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  long_term_memory_exclusions {
    enabled = true
    semantic {
      enabled = true
      prompt  = terraform_data.prompt.output
    }
  }
}
