---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_active_active_subscription_peering"
description: |-
 Active-Active subscription VPC peering resource in the Redis Cloud Terraform provider.
---

# Resource: rediscloud_active_active_subscription_peering

Creates an AWS or GCP VPC peering for an existing Redis Enterprise Cloud Active-Active Subscription, allowing access to your subscription databases as if they were on the same network.

For AWS, peering should be accepted by the other side.
For GCP, the opposite peering request should be submitted.

## Example Usage - AWS

The following example shows how an Active-Active subscription can be peered with an AWS VPC using the rediscloud and AWS providers.

```hcl
resource "rediscloud_active_active_subscription" "subscription-resource" {
  // ...
}

resource "rediscloud_active_active_subscription_peering" "peering-resource" {
   subscription_id = rediscloud_active_active_subscription.subscription-resource.id
   source_region = "us-east-1"
   destination_region = "eu-west-2"
   aws_account_id = "123456789012"
   vpc_id = "vpc-01234567890"
   vpc_cidr = "10.0.10.0/24"
}

resource "aws_vpc_peering_connection_accepter" "aws-peering-resource" {
  vpc_peering_connection_id = rediscloud_active_active_subscription_peering.peering-resource.aws_peering_id
  auto_accept               = true
}
```

## Example Usage - GCP

The following example shows how an Active-Active subscription can be peered with a GCP project network using the rediscloud and google providers.
The example HCL locates the network details and creates/accepts the vpc peering connection through the Google provider.   

```hcl
resource "rediscloud_active_active_subscription" "subscription-resource" {
  // ...
}

data "google_compute_network" "network" {
  project = "my-gcp-project"
  name = "my-gcp-vpc"
}

resource "rediscloud_active_active_subscription_peering" "peering-resource" {
  subscription_id = rediscloud_active_active_subscription.subscription-resource.id
  source_region = "us-central1"
  provider_name = "GCP"
  gcp_project_id = data.google_compute_network.network.project
  gcp_network_name = data.google_compute_network.network.name
}

resource "google_compute_network_peering" "gcp-peering-resource" {
  name         = "peering-gcp-example"
  network      = data.google_compute_network.network.self_link
  peer_network = "https://www.googleapis.com/compute/v1/projects/${rediscloud_active_active_subscription_peering.peering-resource.gcp_redis_project_id}/global/networks/${rediscloud_active_active_subscription_peering.example.gcp_redis_network_name}"
}
```

## Argument Reference

The following arguments are supported:

* `provider_name` - (Optional) The cloud provider to use with the vpc peering, (either `AWS` or `GCP`). Default: ‘AWS’. **Modifying this attribute will force creation of a new resource.**
* `subscription_id` - (Required) A valid Active-Active subscription predefined in the current account. **Modifying this attribute will force creation of a new resource.**
* `source_region` -	(Required) Name of the region to create the VPC peering from. **Modifying this attribute will force creation of a new resource.**


**AWS ONLY:**
* `aws_account_id` - (Required) AWS account ID that the VPC to be peered lives in. **Modifying this attribute will force creation of a new resource.**
* `destination_region` - (Required) Name of the region to create the VPC peering to. **Modifying this attribute will force creation of a new resource.**
* `vpc_id` - (Required) Identifier of the VPC to be peered. **Modifying this attribute will force creation of a new resource.**
* `vpc_cidr` - (Optional) CIDR range of the VPC to be peered. Either this or `vpc_cidrs` must be specified. **Modifying this attribute will force creation of a new resource.**
* `vpc_cidrs` - (Optional) CIDR ranges of the VPC to be peered. Either this or `vpc_cidr` must be specified. **Modifying this attribute will force creation of a new resource.**

**GCP ONLY:**
* `gcp_project_id` - (Required) GCP project ID that the VPC to be peered lives in. **Modifying this attribute will force creation of a new resource.**
* `gcp_network_name` - (Required) The name of the network to be peered. **Modifying this attribute will force creation of a new resource.**

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 10 mins) Used when creating the peering connection
* `delete` - (Defaults to 10 mins) Used when deleting the peering connection

## Attribute reference

* `status` is set to the current status of the peering - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`.

**AWS ONLY:**

* `aws_peering_id` Identifier of the AWS cloud peering

**GCP ONLY:**

* `gcp_redis_project_id` Identifier of the Redis Enterprise Cloud GCP project to be peered
* `gcp_redis_network_name` The name of the Redis Enterprise Cloud network to be peered
* `gcp_peering_id` Identifier of the cloud peering

## Import

`rediscloud_active_active_subscription_peering` can be imported using the ID of the Active-Active subscription and the ID of the peering connection, e.g.

```
$ terraform import rediscloud_active_active_subscription_peering.peering-resource 12345678/1234
```
