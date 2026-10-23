variable "llm_model" { type = string }
variable "embedding_model" { type = string }
variable "key_version" {
  type    = number
  default = 1
}
variable "llm_api_key" {
  type      = string
  sensitive = true
}
variable "embedding_api_key" {
  type      = string
  sensitive = true
}

resource "rediscloud_agent_memory" "test" {
  name                       = "${var.name}-memory"
  database_id                = rediscloud_subscription_database.test[0].db_id
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300

  llm {
    provider = "openai"
    model    = var.llm_model
    credentials {
      type            = "apiKey"
      api_key         = var.llm_api_key
      api_key_version = var.key_version
    }
  }

  embedding {
    provider = "openai"
    model    = var.embedding_model
    credentials {
      type            = "apiKey"
      api_key         = var.embedding_api_key
      api_key_version = var.key_version
    }
  }
}

resource "rediscloud_agent_memory_api_key" "test" {
  store_id = rediscloud_agent_memory.test.id
  name     = "${var.name}-runtime"
}
