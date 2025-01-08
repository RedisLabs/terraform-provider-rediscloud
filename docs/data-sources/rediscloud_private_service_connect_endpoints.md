---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_private_service_connect_endpoints"
description: |-
  Private Service Connect Endpoints data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_private_service_connect_endpoints

The Private Service Connect Endpoints data source allows access to an available the endpoints within your Redis Enterprise Subscription.

## Example Usage

```hcl
data "rediscloud_private_service_connect_endpoints" "example" {
  subscription_id = "1234"
  private_service_connect_service_id = 5678
}

output "rediscloud_endpoints" {
  value = data.rediscloud_private_service_connect.example.endpoints
}
```

## Argument Reference

* `subscription_id` - (Required) The ID of a Pro subscription
* `private_service_connect_service_id` - (Required) The ID of the Private Service Connect Service relative to the associated subscription

## Attribute Reference

* `endpoints` - List of Private Service Connect endpoints, documented below

The `endpoints` object has these attributes:

* `private_service_connect_endpoint_id` - The ID of the Private Service Connect endpoint
* `gcp_project_id` - The Google Cloud Project ID
* `gcp_vpc_name` - The GCP VPC name
* `gcp_vpc_subnet_name` - The GCP Subnet name
* `endpoint_connection_name` - The endpoint connection name
* `status` - The endpoint status
* `service_attachments` - The 40 service attachments that are created for the Private Service Connect endpoint, documented below

The `service_attachments` object has these attributes:

* `name` - Name of the service attachment
* `dns_record` - DNS record for the service attachment
* `ip_address_name` - IP address name for the service attachment
* `forwarding_rule_name` - Name of the forwarding rule for the service attachment
