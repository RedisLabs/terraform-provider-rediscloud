locals {
  name           = "__NAME__"
  gcp_project_id = "__GCP_PROJECT_ID__"
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "google_kms_key_ring" "cmk" {
  project  = local.gcp_project_id
  name     = "${local.name}-keyring"
  location = "europe"
}

resource "google_kms_crypto_key" "cmk" {
  name     = "${local.name}-key"
  key_ring = google_kms_key_ring.cmk.id
}

resource "rediscloud_active_active_subscription" "example" {
  name                         = local.name
  payment_method_id            = data.rediscloud_payment_method.card.id
  customer_managed_key_enabled = true
  cloud_provider               = "GCP"

  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "europe-west1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "europe-west2"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}
