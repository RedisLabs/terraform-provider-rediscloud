resource "rediscloud_agent_memory" "example" {
  name                       = "paired-model-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300

  llm {
    provider = "openai"
    model    = "gpt-4o-mini"
    credentials {
      type            = "apiKey"
      api_key         = "FAKE-MODEL-KEY"
      api_key_version = 1
    }
  }

  embedding {
    provider = "openai"
    model    = "text-embedding-3-small"
    credentials {
      type            = "apiKey"
      api_key         = "FAKE-MODEL-KEY"
      api_key_version = 1
    }
  }
}
