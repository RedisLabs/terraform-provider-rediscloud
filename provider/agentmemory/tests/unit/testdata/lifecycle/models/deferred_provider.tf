variable "provider_input" {
  type    = string
  default = "fake-provider"
}

resource "terraform_data" "provider" { input = var.provider_input }
resource "rediscloud_agent_memory" "example" {
  name                       = "selector-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  llm {
    provider = tostring(terraform_data.provider.output)
    model    = "fake-llm"
    credentials {
      type            = "apiKey"
      api_key         = "FAKE-STATIC"
      api_key_version = 1
    }
  }
  embedding {
    provider = "fake-provider"
    model    = "fake-embedding"
    credentials {
      type            = "apiKey"
      api_key         = "FAKE-STATIC"
      api_key_version = 1
    }
  }

}
