resource "rediscloud_iris_identity_api_key" "key" {
  name = "invalid-key"
  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = "store-1"
    actions       = ["read"]
    constraints   = { schema = "unsupported" }
  }
}
