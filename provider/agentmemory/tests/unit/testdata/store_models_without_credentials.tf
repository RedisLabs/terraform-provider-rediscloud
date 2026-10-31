variable "name" {
  type = string
}

variable "llm_model" {
  type    = string
  default = "gpt-4o-mini"
}

variable "embedding_model" {
  type    = string
  default = "text-embedding-3-small"
}

variable "long_ttl" {
  type    = number
  default = 31536000
}

resource "rediscloud_agent_memory" "example" {
  name                       = var.name
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = var.long_ttl
  extraction_cadence_seconds = 300

  llm {
    provider = "openai"
    model    = var.llm_model
  }

  embedding {
    provider = "openai"
    model    = var.embedding_model
  }
}
