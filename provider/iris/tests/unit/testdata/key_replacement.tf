variable "identity_url" { type = string }
variable "keep_old" { default = true }
variable "create_next" { default = false }
variable "use_next" { default = false }

provider "rediscloud" {
  iris_identity_url   = var.identity_url
  iris_identity_token = "identity-token"
}

resource "rediscloud_iris_identity_api_key" "old" {
  count = var.keep_old ? 1 : 0
  name  = "application-old"
  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = "store-1"
    actions       = ["read"]
  }
}

resource "rediscloud_iris_identity_api_key" "next" {
  count = var.create_next ? 1 : 0
  name  = "application-next"
  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = "store-1"
    actions       = ["read"]
  }
}

output "active_token" {
  value     = var.use_next ? rediscloud_iris_identity_api_key.next[0].token : rediscloud_iris_identity_api_key.old[0].token
  sensitive = true
}
