variable "memory_url" { type = string }
variable "identity_url" { type = string }
variable "name" { type = string }

provider "rediscloud" {
  iris_agent_memory_url   = var.memory_url
  iris_agent_memory_token = "memory-token"
  iris_identity_url       = var.identity_url
  iris_identity_token     = "identity-token"
}

resource "rediscloud_iris_agent_memory" "store" {
  name                         = var.name
  database_id                  = "db-1"
  long_term_memory_ttl_seconds = 3600
  session_summary_enabled      = false
  llm_provider                 = "openai"
  llm_model                    = var.name == "initial" ? "gpt-4o-mini" : "gpt-4.1-mini"
  llm_api_key                  = "fake-${var.name}${var.model_key_suffix}"
  llm_api_key_version          = var.name == "initial" ? 1 : 2
  embedding_provider           = "openai"
  embedding_model              = var.embedding_model
  embedding_api_key            = "fake-${var.name}${var.model_key_suffix}"
  embedding_api_key_version    = var.name == "initial" ? 1 : 2
  exclusions_enabled           = true
  built_in_detectors_enabled   = true
  custom_detectors_enabled     = true
  built_in_detector {
    id      = "email"
    enabled = true

  }
  custom_detector {
    name    = "private"
    enabled = true

    pattern = "secret[0-9]+"
  }
  custom_memory_type {
    name               = "preference"
    description        = "Preferences"
    extraction_prompt  = "Extract preferences"
    extraction_enabled = true
    field {
      name        = "topic"
      description = "Topic"
      type        = "STRING"
    }
  }

}
resource "rediscloud_iris_agent_memory_type" "profile" {
  store_id    = rediscloud_iris_agent_memory.store.id
  name        = "profile"
  description = "Profile"
  field {
    name        = "topic"
    description = "Topic"
    type        = "STRING"
  }
  extraction_prompt  = var.name
  extraction_enabled = true
}
variable "key_name" { default = null }
resource "rediscloud_iris_identity_api_key" "key" {
  name       = var.key_name != null ? var.key_name : var.name
  expires_at = 0
  tenant     = "tenant-1"
  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = "store-1"
    actions       = var.name == "initial" ? ["read"] : ["read", "write"]
  }
  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = "store-2"
    actions       = ["read"]
  }
}
variable "embedding_model" { default = "text-embedding-3-small" }

variable "model_key_suffix" { default = "" }
