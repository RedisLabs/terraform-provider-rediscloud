resource "terraform_data" "ttl" { input = 172800 }
resource "rediscloud_agent_memory" "example" {
  name                       = "unknown-ttl-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  extraction_cadence_seconds = 300
  llm {
    provider = "fake-provider"
    model    = "fake-model"
    credentials {
      type            = "apiKey"
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
