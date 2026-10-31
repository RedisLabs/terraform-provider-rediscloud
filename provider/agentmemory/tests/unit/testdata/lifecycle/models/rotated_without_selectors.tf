variable "embedding_credentials_api_key" {
  type    = string
  default = "FAKE-ROTATED"
}

variable "embedding_credentials_api_key_version" {
  type    = number
  default = 2
}

variable "llm_credentials_api_key" {
  type    = string
  default = "FAKE-ROTATED"
}

variable "llm_credentials_api_key_version" {
  type    = number
  default = 2
}

resource "rediscloud_agent_memory" "example" {
  name                       = "selector-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  llm {
    model = "fake-llm"
    credentials {
      type            = "apiKey"
      api_key         = var.llm_credentials_api_key
      api_key_version = var.llm_credentials_api_key_version
    }
  }
  embedding {
    credentials {
      type            = "apiKey"
      api_key         = var.embedding_credentials_api_key
      api_key_version = var.embedding_credentials_api_key_version
    }
  }

}
