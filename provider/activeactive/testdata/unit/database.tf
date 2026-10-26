variable "subscription_id" {
  type = number
}

variable "name" {
  type = string
}

variable "redis_version" {
  type = string
}

# global_password is pinned rather than left to the API. A config that leaves it null makes every plan
# that changes another attribute also plan this attribute as unknown, which defeats an empty-plan
# assertion about anything else.
resource "test_active_active_subscription_database" "aa_db" {
  subscription_id    = var.subscription_id
  name               = var.name
  dataset_size_in_gb = 1
  global_password    = "test-password"
  redis_version      = var.redis_version
}
