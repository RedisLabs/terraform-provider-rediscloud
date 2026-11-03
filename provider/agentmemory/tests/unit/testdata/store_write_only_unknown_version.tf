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

variable "llm_key" {
  type      = string
  sensitive = true
  ephemeral = true
}

variable "embedding_key" {
  type      = string
  sensitive = true
  ephemeral = true
}

variable "llm_version" {
  type = number
}
variable "embedding_version" {
  type = number
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

    credentials {
      type            = "apiKey"
      api_key         = var.llm_key
      api_key_version = terraform_data.version.output
    }
  }

  embedding {
    provider = "openai"
    model    = var.embedding_model

    credentials {
      type            = "apiKey"
      api_key         = var.embedding_key
      api_key_version = var.embedding_version
    }
  }
}

resource "terraform_data" "version" {
  input = var.llm_version
}
