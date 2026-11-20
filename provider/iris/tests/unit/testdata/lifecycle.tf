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
  name                  = var.name
  database_json         = jsonencode({ databaseId = "db-1" })
  long_term_memory_json = jsonencode({ ttlSeconds = 3600, sessionSummaryEnabled = false })
}
resource "rediscloud_iris_agent_memory_type" "profile" {
  store_id                 = rediscloud_iris_agent_memory.store.id
  name                     = "profile"
  description              = "Profile"
  fields_json              = jsonencode([{ name = "topic", description = "Topic", type = "STRING" }])
  extraction_strategy_json = jsonencode({ prompt = var.name, enabled = true })
}
resource "rediscloud_iris_agent_memory_extractor" "extractor" {
  store_id                 = rediscloud_iris_agent_memory.store.id
  name                     = var.name
  memory_type              = rediscloud_iris_agent_memory_type.profile.name
  source                   = "DOCUMENTS"
  strategy                 = "CUSTOM"
  prompt                   = var.name
  enabled                  = false
  destination_namespace_id = var.name == "initial" ? "ns_55555555555555555555555555555555" : ""
}
resource "rediscloud_iris_identity_api_key" "key" {
  name       = var.name
  expires_at = 0
  grants_json = jsonencode([{
    tenant  = "tenant-1", product = "memory", resourceType = "store", resourceId = "store-1",
    actions = ["read"], constraints = { schema = "test", nested = { enabled = false, count = 0 } }
  }])
}
resource "rediscloud_iris_identity_provider" "oidc" {
  tenant              = "tenant-1"
  name                = "Corporate"
  issuer              = "https://issuer.example.com"
  audience            = var.name
  jwks_uri            = "https://issuer.example.com/keys"
  required_algorithms = ["RS256"]
  token_exchange_json = jsonencode({ grants = "SUBJECT_GRANTS", irisAudiences = ["memory"] })
}
resource "rediscloud_iris_identity_subject_grant_set" "subject" {
  tenant  = "tenant-1"
  issuer  = "https://issuer.example.com"
  subject = "user-1"
}
