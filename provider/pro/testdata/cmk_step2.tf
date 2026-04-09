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

  labels = {
    managed-by = "terraform"
    purpose    = "rediscloud-cmk-test"
  }
}

resource "google_kms_crypto_key_iam_member" "encrypter" {
  crypto_key_id = google_kms_crypto_key.cmk.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${rediscloud_subscription.example.customer_managed_key_redis_service_account}"
}

resource "google_kms_crypto_key_iam_member" "viewer" {
  crypto_key_id = google_kms_crypto_key.cmk.id
  role          = "roles/cloudkms.viewer"
  member        = "serviceAccount:${rediscloud_subscription.example.customer_managed_key_redis_service_account}"
}

resource "rediscloud_subscription" "example" {
  name                         = local.name
  payment_method               = "credit-card"
  payment_method_id            = data.rediscloud_payment_method.card.id
  memory_storage               = "ram"
  customer_managed_key_enabled = true

  cloud_provider {
    provider = "GCP"
    region {
      region                     = "europe-west2"
      networking_deployment_cidr = "10.0.1.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
  }

  customer_managed_key {
    resource_name = google_kms_crypto_key.cmk.id
  }
}
