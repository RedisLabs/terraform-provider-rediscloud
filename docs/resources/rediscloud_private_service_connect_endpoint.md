---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_private_service_connect"
description: |-
  Private Service Connect Endpoint resource fo Pro Subscription in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_private_service_connect

Manages a Private Service Connect to a Pro subscription in your Redis Enterprise Cloud Account.

## Example Usage

The example below creates a Private Service Connect Endpoint in a Pro subscription, the respective GCP resources 
and accepts the endpoint. 

Please note that an endpoint can only be accepted after, the forwarding rules in GCP are created.

```hcl
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "subscription" {
  name              = "subscription-name"
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    provider = "GCP"
    region {
      region                     = var.gcp_region
      networking_deployment_cidr = "10.0.1.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb           = 15
    quantity                     = 1
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 20000
  }
}

locals {
  service_attachment_count = 40 # Each rediscloud_private_service_connect_endpoint will have exactly 40 service attachments
}

resource "rediscloud_private_service_connect" "service" {
  subscription_id = rediscloud_subscription.subscription.id
}

resource "rediscloud_private_service_connect_endpoint" "endpoint" {
  subscription_id                    = rediscloud_subscription.subscription.id
  private_service_connect_service_id = rediscloud_private_service_connect.service.private_service_connect_service_id

  gcp_project_id           = var.gcp_project_id
  gcp_vpc_name             = var.gcp_vpc_name
  gcp_vpc_subnet_name      = var.gcp_subnet_name
  endpoint_connection_name = "redis-${rediscloud_subscription.subscription.id}"
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
  name         = rediscloud_private_service_connect_endpoint.endpoint.service_attachments[count.index].ip_address_name
  subnetwork   = data.google_compute_subnetwork.subnet.id
  address_type = "INTERNAL"
  region       = var.gcp_region
}

resource "google_compute_forwarding_rule" "default" {
  count = local.service_attachment_count

  name                  = rediscloud_private_service_connect_endpoint.endpoint.service_attachments[count.index].forwarding_rule_name
  project               = var.gcp_project_id
  region                = var.gcp_region
  ip_address            = google_compute_address.default[count.index].id
  network               = var.gcp_vpc_name
  target                = rediscloud_private_service_connect_endpoint.endpoint.service_attachments[count.index].name
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
  rule_name       = "${rediscloud_private_service_connect_endpoint.endpoint.service_attachments[count.index].forwarding_rule_name}-${var.gcp_region}-rule"
  dns_name        = rediscloud_private_service_connect_endpoint.endpoint.service_attachments[count.index].dns_record

  local_data {
    local_datas {
      name    = rediscloud_private_service_connect_endpoint.endpoint.service_attachments[count.index].dns_record
      type    = "A"
      ttl     = 300
      rrdatas = [google_compute_address.default[count.index].address]
    }
  }
}

resource "rediscloud_private_service_connect_endpoint_accepter" "accepter" {
  subscription_id                     = rediscloud_subscription.subscription.id
  private_service_connect_service_id  = rediscloud_private_service_connect.service.private_service_connect_service_id
  private_service_connect_endpoint_id = rediscloud_private_service_connect_endpoint.endpoint.private_service_connect_endpoint_id

  action = "accept"

  depends_on = [google_compute_forwarding_rule.default]
}

```

## Argument Reference

* `subscription_id` - (Required) The ID of the Pro subscription to attach **Modifying this attribute will force creation of a new resource.**
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
`rediscloud_private_service_connect_endpoint` can be imported using the ID of the subscription and the ID of the Private Service Connect in the format {subscription ID/private service connect ID/private service connect endpoint ID}, e.g.

```
$ terraform import rediscloud_private_service_connect_endpoint.id 1000/123456/654321
```
