---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_private_service_connect_endpoint"
description: |-
  Private Service Connect Endpoint resource for Active-Active Subscription in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_private_service_connect

Manages a Private Service Connect to an Active-Active Subscription in your Redis Enterprise Cloud Account.

## Example Usage

The example below creates a Private Service Connect Endpoint in an Active-Active subscription, the respective GCP resources 
and accepts the endpoint for the `us-central1` region. 

Please note that an endpoint can only be accepted after, the forwarding rules in GCP are created.

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "subscription" {
  name              = "subscription-name"
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

locals {
  service_attachment_count = 40 # Each rediscloud_active_active_private_service_connect_endpoint will have exactly 40 service attachments
  region_id                = one([for r in rediscloud_active_active_subscription_regions.regions.region : r.region_id if r.region == var.gcp_region])
}

resource "rediscloud_active_active_private_service_connect" "service" {
  subscription_id = rediscloud_active_active_subscription.subscription.id
  region_id       = local.region_id
}

resource "rediscloud_active_active_private_service_connect_endpoint" "endpoint" {
  subscription_id                    = rediscloud_active_active_subscription.subscription.id
  region_id                          = local.region_id
  private_service_connect_service_id = rediscloud_active_active_private_service_connect.service.private_service_connect_service_id

  gcp_project_id           = var.gcp_project_id
  gcp_vpc_name             = var.gcp_vpc_name
  gcp_vpc_subnet_name      = var.gcp_subnet_name
  endpoint_connection_name = "redis-${rediscloud_active_active_subscription.subscription.id}"
}

data "google_compute_network" "network" {
  project = var.gcp_project_id
  name    = var.gcp_vpc_name
}

data "google_compute_subnetwork" "subnet" {
  project = var.gcp_project_id
  name    = var.gcp_subnet_name
  region  = var.gcp_region
}

resource "google_compute_address" "default" {
  count = local.service_attachment_count

  project      = var.gcp_project_id
  name         = rediscloud_active_active_private_service_connect_endpoint.endpoint.service_attachments[count.index].ip_address_name
  subnetwork   = data.google_compute_subnetwork.subnet.id
  address_type = "INTERNAL"
  region       = var.gcp_region
}

resource "google_compute_forwarding_rule" "default" {
  count = local.service_attachment_count

  name                  = rediscloud_active_active_private_service_connect_endpoint.endpoint.service_attachments[count.index].forwarding_rule_name
  project               = var.gcp_project_id
  region                = var.gcp_region
  ip_address            = google_compute_address.default[count.index].id
  network               = var.gcp_vpc_name
  target                = rediscloud_active_active_private_service_connect_endpoint.endpoint.service_attachments[count.index].name
  load_balancing_scheme = ""
}

resource "google_dns_response_policy" "redis_response_policy" {
  response_policy_name = "redis-${var.gcp_vpc_name}"
  project              = var.gcp_project_id

  networks {
    network_url = data.google_compute_network.network.id
  }
}

resource "google_dns_response_policy_rule" "redis_response_policy_rules" {
  count = local.service_attachment_count

  project         = var.gcp_project_id
  response_policy = google_dns_response_policy.redis_response_policy.response_policy_name
  rule_name       = "${rediscloud_active_active_private_service_connect_endpoint.endpoint.service_attachments[count.index].forwarding_rule_name}-${var.gcp_region}-rule"
  dns_name        = rediscloud_active_active_private_service_connect_endpoint.endpoint.service_attachments[count.index].dns_record

  local_data {
    local_datas {
      name    = rediscloud_active_active_private_service_connect_endpoint.endpoint.service_attachments[count.index].dns_record
      type    = "A"
      ttl     = 300
      rrdatas = [google_compute_address.default[count.index].address]
    }
  }
}

resource "rediscloud_active_active_private_service_connect_endpoint_accepter" "accepter" {
  subscription_id                     = rediscloud_active_active_subscription.subscription.id
  region_id                           = local.region_id
  private_service_connect_service_id  = rediscloud_active_active_private_service_connect.service.private_service_connect_service_id
  private_service_connect_endpoint_id = rediscloud_active_active_private_service_connect_endpoint.endpoint.private_service_connect_endpoint_id

  action = "accept"

  depends_on = [google_compute_forwarding_rule.default]
}

```

## Example Usage with Redis Private Service Connect Module

The example below creates a Private Service Connect Endpoint in an Active-Active subscription using the [Redis Cloud PSC Terraform module](https://registry.terraform.io/modules/RedisLabs/private-service-connect/rediscloud/latest).
We recommend using the module as it simplifies the Terraform configuration.

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_active_active_subscription" "subscription" {
  name              = "subscription-name"
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

locals {
  region_id = one([for r in rediscloud_active_active_subscription_regions.regions.region : r.region_id if r.region == var.gcp_region])
}

module "private_service_connect" {
  source = "RedisLabs/private-service-connect/rediscloud"

  rediscloud_subscription_type = "active-active"
  rediscloud_subscription_id   = rediscloud_active_active_subscription.subscription.id
  rediscloud_region_id         = local.region_id

  gcp_region = var.gcp_region

  endpoints = [
    {
      gcp_project_id           = var.gcp_project_id
      gcp_vpc_name             = var.gcp_vpc_name
      gcp_vpc_subnetwork_name  = var.gcp_subnet_name
      gcp_response_policy_name = var.gcp_response_policy_name
    }
  ]
}

```

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro subscription to attach **Modifying this attribute will force creation of a new resource.**
* `region_id` - (Required) The ID of the region, as created by the API **Modifying this attribute will force creation of a new resource.**
* `private_service_connect_service_id` - (Required) The ID of the Private Service Connect Service relative to the associated subscription **Modifying this attribute will force creation of a new resource.**
* `gcp_project_id` - (Required) The Google Cloud Project ID **Modifying this attribute will force creation of a new resource.**
* `gcp_vpc_name` - (Required) The GCP VPC Network name **Modifying this attribute will force creation of a new resource.**
* `gcp_vpc_subnet_name` - (Required) The GCP Subnet name **Modifying this attribute will force creation of a new resource.**
* `endpoint_connection_name` - (Required) The endpoint connection name prefix. This prefix that will be used to create the Private Service Connect endpoint in your Google Cloud account **Modifying this attribute will force creation of a new resource.**

## Attribute Reference

* `private_service_connect_endpoint_id` - The ID of the Private Service Connect Endpoint
* `service_attachments` - The 40 service attachments that are created for the Private Service Connect endpoint, documented below

The `service_attachments` object has these attributes:

* `name` - Name of the service attachment
* `dns_record` - DNS record for the service attachment
* `ip_address_name` - IP address name for the service attachment
* `forwarding_rule_name` - Name of the forwarding rule for the service attachment

## Import

`rediscloud_active_active_private_service_connect_endpoint` can be imported using the ID of the Active-Active subscription, the region ID and the ID of the Private Service Connect in the format {subscription ID/region ID/private service connect ID//private service connect endpoint ID}, e.g.

```
$ terraform import rediscloud_active_active_private_service_connect_endpoint.id 1000/1/123456/654321
```
