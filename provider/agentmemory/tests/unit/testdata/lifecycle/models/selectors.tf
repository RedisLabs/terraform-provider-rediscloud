variable "embedding_credentials_api_key" {
  type    = string
  default = "FAKE-STATIC"
}

variable "embedding_credentials_api_key_version" {
  type    = number
  default = 1
}

variable "embedding_model" {
  type    = string
  default = "fake-embedding"
}

variable "embedding_provider" {
  type    = string
  default = "fake-provider"
}

variable "llm_credentials_api_key" {
  type    = string
  default = "FAKE-STATIC"
}

variable "llm_credentials_api_key_version" {
  type    = number
  default = 1
}

variable "llm_model" {
  type    = string
  default = "fake-llm"
}

variable "llm_provider" {
  type    = string
  default = "fake-provider"
}

resource "rediscloud_agent_memory" "example" {
  name                       = "selector-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  llm {
    provider = var.llm_provider
    model    = var.llm_model
    credentials {
      type            = "apiKey"
      api_key         = var.llm_credentials_api_key
      api_key_version = var.llm_credentials_api_key_version
    }
  }
  embedding {
    provider = var.embedding_provider
    model    = var.embedding_model
    credentials {
      type            = "apiKey"
      api_key         = var.embedding_credentials_api_key
      api_key_version = var.embedding_credentials_api_key_version
    }
  }

}
