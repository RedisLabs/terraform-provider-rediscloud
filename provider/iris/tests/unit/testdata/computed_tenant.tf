variable "identity_url" { type = string }
variable "tenant" { type = string }
variable "computed_tenant" { default = true }
variable "store_id" { default = "store-1" }
variable "tenant_trigger" { default = null }

provider "rediscloud" {
  iris_identity_url   = var.identity_url
  iris_identity_token = "identity-token"
}

resource "terraform_data" "tenant" {
  input            = var.tenant
  triggers_replace = var.tenant_trigger
}

resource "terraform_data" "store" {
  input = var.store_id
}

resource "rediscloud_iris_identity_api_key" "key" {
  name   = "computed-tenant"
  tenant = var.computed_tenant ? terraform_data.tenant.output : var.tenant
  grant {
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = terraform_data.store.output
    actions       = ["read"]
  }
}
