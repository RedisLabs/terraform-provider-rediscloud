resource "rediscloud_iris_identity_api_key" "example" {
  name       = "application-runtime"
  expires_at = 0
  tenant     = "your-tenant"

  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = rediscloud_iris_agent_memory.example.id
    actions       = ["read", "write"]
  }
}
