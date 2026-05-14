variable "subscription_name" {
  type = string
}

variable "gcp_project_id" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type         = "Visa"
  last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "subscription" {
  name              = var.subscription_name
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider    = "GCP"

  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "us-central1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "europe-west1"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "database" {
  subscription_id         = rediscloud_active_active_subscription.subscription.id
  name                    = "db"
  memory_limit_in_gb      = 1
  global_data_persistence = "aof-every-1-second"
  global_password         = "some-password"
}

resource "rediscloud_active_active_subscription_regions" "regions" {
  subscription_id = rediscloud_active_active_subscription.subscription.id

  region {
    region                     = "us-central1"
    networking_deployment_cidr = "192.168.0.0/24"
    database {
      database_id                       = rediscloud_active_active_subscription_database.database.db_id
      database_name                     = rediscloud_active_active_subscription_database.database.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second  = 1000
    }
  }

  region {
    region                     = "europe-west1"
    networking_deployment_cidr = "10.0.1.0/24"
    database {
      database_id                       = rediscloud_active_active_subscription_database.database.db_id
      database_name                     = rediscloud_active_active_subscription_database.database.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second  = 1000
    }
  }
}

resource "rediscloud_active_active_private_service_connect" "psc" {
  subscription_id = rediscloud_active_active_subscription.subscription.id
  region_id       = one([for r in rediscloud_active_active_subscription_regions.regions.region : r.region_id if r.region == "us-central1"])
}

resource "rediscloud_active_active_private_service_connect_endpoint" "psce" {
  subscription_id                    = rediscloud_active_active_subscription.subscription.id
  region_id                          = one([for r in rediscloud_active_active_subscription_regions.regions.region : r.region_id if r.region == "us-central1"])
  private_service_connect_service_id = rediscloud_active_active_private_service_connect.psc.private_service_connect_service_id
  gcp_project_id                     = var.gcp_project_id
  gcp_vpc_name                       = google_compute_network.network.name
  gcp_vpc_subnet_name                = google_compute_subnetwork.subnet.name
  endpoint_connection_name           = "redis-${rediscloud_active_active_subscription.subscription.id}"
}

resource "google_compute_network" "network" {
  project                 = var.gcp_project_id
  name                    = var.subscription_name
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  project       = var.gcp_project_id
  name          = var.subscription_name
  ip_cidr_range = "192.168.1.0/24"
  region        = "us-central1"
  network       = google_compute_network.network.id
}

locals {
  service_attachment_count = 1
}

resource "google_compute_address" "default" {
  count = local.service_attachment_count

  project      = var.gcp_project_id
  name         = rediscloud_active_active_private_service_connect_endpoint.psce.service_attachments[count.index].ip_address_name
  subnetwork   = google_compute_subnetwork.subnet.id
  address_type = "INTERNAL"
  region       = "us-central1"
}

resource "google_compute_forwarding_rule" "default" {
  count = local.service_attachment_count

  name                  = rediscloud_active_active_private_service_connect_endpoint.psce.service_attachments[count.index].forwarding_rule_name
  project               = var.gcp_project_id
  region                = "us-central1"
  ip_address            = google_compute_address.default[count.index].id
  network               = google_compute_network.network.name
  target                = rediscloud_active_active_private_service_connect_endpoint.psce.service_attachments[count.index].name
  load_balancing_scheme = ""
}

resource "rediscloud_active_active_private_service_connect_endpoint_accepter" "accepter" {
  subscription_id                     = rediscloud_active_active_subscription.subscription.id
  region_id                           = one([for r in rediscloud_active_active_subscription_regions.regions.region : r.region_id if r.region == "us-central1"])
  private_service_connect_service_id  = rediscloud_active_active_private_service_connect.psc.private_service_connect_service_id
  private_service_connect_endpoint_id = rediscloud_active_active_private_service_connect_endpoint.psce.private_service_connect_endpoint_id

  action = "accept"

  depends_on = [google_compute_forwarding_rule.default]
}
