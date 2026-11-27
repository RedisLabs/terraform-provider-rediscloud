variable "identity_url" { type = string }
variable "tenant" { type = string }

provider "rediscloud" {
  iris_identity_url   = var.identity_url
  iris_identity_token = "identity-token"
}

resource "rediscloud_iris_identity_api_key" "key" {
  name   = "multi-grant"
  tenant = var.tenant
  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = "store-1"
    actions       = ["read"]
  }
  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = "store-2"
    actions       = ["write"]
  }
}
