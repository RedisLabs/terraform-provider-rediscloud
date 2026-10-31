resource "rediscloud_agent_memory" "example" {
  name                  = "store"
  database_id           = 123
  long_term_ttl_seconds = 31536000

  llm {
    provider = "openai"
    model    = "gpt-4o-mini"

    credentials {
      type            = "apiKey"
      api_key         = "llm-secret"
      api_key_version = 1
    }
  }
}
