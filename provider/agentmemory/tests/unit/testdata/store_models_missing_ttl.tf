resource "rediscloud_agent_memory" "example" {
  name        = "store"
  database_id = 123

  llm {
    provider = "openai"
    model    = "gpt-4o-mini"

    credentials {
      type            = "apiKey"
      api_key         = "llm-secret"
      api_key_version = 1
    }
  }

  embedding {
    provider = "openai"
    model    = "text-embedding-3-small"

    credentials {
      type            = "apiKey"
      api_key         = "embedding-secret"
      api_key_version = 1
    }
  }
}
