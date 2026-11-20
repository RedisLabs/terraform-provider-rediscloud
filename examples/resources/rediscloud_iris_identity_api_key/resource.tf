resource "rediscloud_iris_identity_api_key" "example" {
  name       = "application-runtime"
  expires_at = 0
  grants_json = jsonencode([{
    tenant       = "your-tenant"
    product      = "memory"
    resourceType = "store"
    resourceId   = rediscloud_iris_agent_memory.example.id
    actions      = ["read", "write"]
  }])
}
