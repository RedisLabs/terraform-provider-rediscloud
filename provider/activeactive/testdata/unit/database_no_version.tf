variable "subscription_id" {
  type = number
}

variable "name" {
  type = string
}

# redis_version is absent, so the database is created at whatever version the API picks.
resource "test_active_active_subscription_database" "aa_db" {
  subscription_id    = var.subscription_id
  name               = var.name
  dataset_size_in_gb = 1
  global_password    = "test-password"
}
