variable "subscription_id" {
  type = number
}

variable "name" {
  type = string
}

variable "redis_version" {
  type = string
}

# global_password is absent, which is the default configuration where the API generates one.
resource "test_active_active_subscription_database" "aa_db" {
  subscription_id    = var.subscription_id
  name               = var.name
  dataset_size_in_gb = 1
  redis_version      = var.redis_version
}
