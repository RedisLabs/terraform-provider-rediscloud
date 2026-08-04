variable "access_key_id" {
  type = string
}

variable "access_secret_key" {
  type = string
}

variable "console_username" {
  type = string
}

variable "console_password" {
  type = string
}

variable "name" {
  type = string
}

variable "provider_type" {
  type = string
}

variable "sign_in_login_url" {
  type = string
}

resource "rediscloud_cloud_account" "test" {
  access_key_id     = var.access_key_id
  access_secret_key = var.access_secret_key
  console_username  = var.console_username
  console_password  = var.console_password
  name              = var.name
  provider_type     = var.provider_type
  sign_in_login_url = var.sign_in_login_url
}
