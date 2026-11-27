resource "rediscloud_iris_identity_api_key" "key" {
  name = "grant-tenant"
  grant {
    tenant        = "tenant-one"
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = "store-1"
    actions       = ["read"]
  }
}
