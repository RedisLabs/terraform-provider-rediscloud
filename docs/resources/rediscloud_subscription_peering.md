---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription_peering"
description: |-
  Subscription VPC peering resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_subscription_peering

Creates an AWS or GCP VPC peering for an existing Redis Enterprise Cloud Subscription, allowing access to your subscription databases as if they were on the same network.

For AWS, peering should be accepted by the other side.
For GCP, the opposite peering request should be submitted.

## Example Usage - AWS

The following example shows how a subscription can be peered with a AWS VPC.
The terraform output value shows how an example `aws ec2 accept-vpc-peering-connection` command can be returned for the user to accept the peering request.

```hcl
resource "rediscloud_subscription" "example" {
  // ...
}

resource "rediscloud_subscription_peering" "example" {
   subscription_id = rediscloud_subscription.example.id
   region = "eu-west-1"
   aws_account_id = "123456789012"
   vpc_id = "vpc-01234567890"
   vpc_cidr = "10.0.0.0/8"
}

output "aws_peering_accept_cmd" {
  value = <<-EOF
  aws ec2 accept-vpc-peering-connection --vpc-peering-connection-id ${rediscloud_subscription_peering.example.aws_peering_id}
  EOF
}
```

## Example Usage - GCP

The following example shows how a subscription can be peered with a GCP project network using the rediscloud and google providers.
The example HCL locates the network details and creates/accepts the vpc peering connection through the Google provider.   

```hcl
resource "rediscloud_subscription" "example" {
  // ...
}

resource "rediscloud_subscription_peering" "example" {
   subscription_id = rediscloud_subscription.example.id
   provider = "GCP"
   gcp_project_id = "cloud-api-123456"
   gcp_network_name = "cloud-api-vpc-peering-example"
}

data "google_compute_network" "network" {
  project = "my-gcp-project"
  name = "my-gcp-vpc"
}

resource "rediscloud_subscription_peering" "example-peering" {
  subscription_id = rediscloud_subscription.example.id
  provider_name = "GCP"
  gcp_project_id = data.google_compute_network.network.project
  gcp_network_name = data.google_compute_network.network.name
}

resource "google_compute_network_peering" "example-peering" {
  name         = "peering-gcp-example"
  network      = data.google_compute_network.network.self_link
  peer_network = "https://www.googleapis.com/compute/v1/projects/${rediscloud_subscription_peering.example.gcp_redis_project_id}/global/networks/${rediscloud_subscription_peering.example.gcp_redis_network_name}"
}
```

## Argument Reference

The following arguments are supported:

* `provider_name` - (Optional) The cloud provider to use with the vpc peering, (either `AWS` or `GCP`). Default: ‘AWS’
* `subscription_id` - (Required) A valid subscription predefined in the current account

**AWS ONLY:**
* `aws_account_id` - (Required AWS) AWS account ID that the VPC to be peered lives in
* `region` - (Required AWS) AWS Region that the VPC to be peered lives in
* `vpc_id` - (Required AWS) Identifier of the VPC to be peered
* `vpc_cidr` - (Required AWS) CIDR range of the VPC to be peered 

**GCP ONLY:**
* `gcp_project_id` - (Required GCP) GCP project ID that the VPC to be peered lives in
* `gcp_network_name` - (Required GCP) The name of the network to be peered

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 10 mins) Used when creating the peering connection
* `delete` - (Defaults to 10 mins) Used when deleting the peering connection

## Attribute reference

* `status` is set to the current status of the account - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`.

**AWS ONLY:**

* `aws_peering_id` Identifier of the AWS cloud peering

**GCP ONLY:**

* `gcp_redis_project_id` Identifier of the Redis Enterprise Cloud GCP project to be peered
* `gcp_redis_network_name` The name of the Redis Enterprise Cloud network to be peered
* `gcp_peering_id` Identifier of the cloud peering

## Import

`rediscloud_subscription_peering` can be imported using the ID of the subscription and the ID of the peering connection, e.g.

```
$ terraform import rediscloud_subscription_peering.example 12345678/1234
```
