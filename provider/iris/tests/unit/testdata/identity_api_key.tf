variable "identity_url" { type = string }
variable "product" { default = "memory" }
variable "resource_type" { default = "mem-store" }
variable "name" { default = "local-environment-test" }
variable "actions" { default = ["read"] }
variable "expires_at" { default = 0 }
variable "user_id" { default = null }
variable "tenant" { default = null }
provider "rediscloud" {
  iris_identity_url   = var.identity_url
  iris_identity_token = "identity-token"
}
resource "rediscloud_iris_identity_api_key" "key" {
  name       = var.name
  user_id    = var.user_id
  expires_at = var.expires_at == 0 ? null : var.expires_at
  tenant     = var.tenant
  grant {
    product       = var.product
    resource_type = var.resource_type
    resource_id   = "store-1"
    actions       = var.actions
  }
}
