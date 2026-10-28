variable "key" {
  type = string
}

variable "key_version" {
  type = number
}

resource "terraform_data" "key" { input = var.key }
resource "rediscloud_agent_memory" "example" {
  name                       = "unknown-key-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  llm {
    provider = "fake-provider"
    model    = "fake-llm"
    credentials {
      type            = "apiKey"
      api_key         = "FAKE-STATIC"
      api_key_version = var.key_version
    }
  }
  embedding {
    provider = "fake-provider"
    model    = "fake-embedding"
    credentials {
      type            = "apiKey"
      api_key         = tostring(terraform_data.key.output)
      api_key_version = var.key_version
    }
  }
}
