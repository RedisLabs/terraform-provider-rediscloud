variable "memory_url" { type = string }
variable "model_providers" {
  type = map(string)
  default = {
    llm            = "redis_local"
    embedding      = "redis_local"
    classification = "redis_local"
  }
}
variable "models" {
  type = map(string)
  default = {
    llm            = "redis-llm-v1"
    embedding      = "redis-embed-v1"
    classification = "redis-llm-v1"
  }
}
variable "keys" {
  type      = map(string)
  default   = {}
  sensitive = true
  ephemeral = true
}
variable "versions" {
  type    = map(number)
  default = {}
}

provider "rediscloud" {
  iris_agent_memory_url   = var.memory_url
  iris_agent_memory_token = "memory-token"
}

resource "rediscloud_iris_agent_memory" "store" {
  name                           = "selected-models"
  database_id                    = "db-1"
  llm_provider                   = lookup(var.model_providers, "llm", null)
  llm_model                      = lookup(var.models, "llm", null)
  llm_api_key                    = lookup(var.keys, "llm", null)
  llm_api_key_version            = lookup(var.versions, "llm", null)
  embedding_provider             = lookup(var.model_providers, "embedding", null)
  embedding_model                = lookup(var.models, "embedding", null)
  embedding_api_key              = lookup(var.keys, "embedding", null)
  embedding_api_key_version      = lookup(var.versions, "embedding", null)
  classification_provider        = lookup(var.model_providers, "classification", null)
  classification_model           = lookup(var.models, "classification", null)
  classification_api_key         = lookup(var.keys, "classification", null)
  classification_api_key_version = lookup(var.versions, "classification", null)
}
