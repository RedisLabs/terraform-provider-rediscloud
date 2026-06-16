terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.5"
    }
  }
}

variable "name" {
  type = string
}

variable "gcp_project_id" {
  type = string
}

variable "maintenance_windows" {
  type = list(object({
    mode = string
    window = list(object({
      start_hour        = number
      duration_in_hours = number
      days              = list(string)
    }))
  }))
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "google_kms_key_ring" "cmk" {
  project  = var.gcp_project_id
  name     = "${var.name}-keyring"
  location = "europe"
}

resource "google_kms_crypto_key" "cmk" {
  name     = "${var.name}-key"
  key_ring = google_kms_key_ring.cmk.id

  labels = {
    managed-by = "terraform"
    purpose    = "rediscloud-cmk-test"
  }
}

resource "google_kms_crypto_key_iam_member" "encrypter" {
  crypto_key_id = google_kms_crypto_key.cmk.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${rediscloud_active_active_subscription.example.customer_managed_key_redis_service_account}"
}

resource "google_kms_crypto_key_iam_member" "viewer" {
  crypto_key_id = google_kms_crypto_key.cmk.id
  role          = "roles/cloudkms.viewer"
  member        = "serviceAccount:${rediscloud_active_active_subscription.example.customer_managed_key_redis_service_account}"
}

resource "rediscloud_active_active_subscription" "example" {
  name                         = var.name
  payment_method_id            = data.rediscloud_payment_method.card.id
  customer_managed_key_enabled = true
  cloud_provider               = "GCP"

  dynamic "maintenance_windows" {
    for_each = var.maintenance_windows
    content {
      mode = maintenance_windows.value.mode
      dynamic "window" {
        for_each = maintenance_windows.value.window
        content {
          start_hour        = window.value.start_hour
          duration_in_hours = window.value.duration_in_hours
          days              = window.value.days
        }
      }
    }
  }

  customer_managed_key {
    resource_name = google_kms_crypto_key.cmk.id
    region        = "europe-west1"
  }

  customer_managed_key {
    resource_name = google_kms_crypto_key.cmk.id
    region        = "europe-west2"
  }

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
