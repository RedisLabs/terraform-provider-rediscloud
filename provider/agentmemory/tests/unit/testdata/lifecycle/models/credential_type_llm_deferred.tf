variable "input" {
  type = string
}

resource "terraform_data" "selector" { input = var.input }
resource "rediscloud_agent_memory" "example" {
  name                       = "credential-type-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  llm {
    provider = "fake-provider"
    model    = "fake-model"
    credentials {
      type            = split(":", tostring(terraform_data.selector.output))[0]
      api_key         = "FAKE-STATIC"
      api_key_version = 1
    }
  }
  embedding {
    provider = "fake-provider"
    model    = "fake-model"
    credentials {
      type            = "apiKey"
      api_key         = "FAKE-STATIC"
      api_key_version = 1
    }
  }

}
