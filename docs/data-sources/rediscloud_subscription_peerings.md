---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription_peerings"
description: |-
  Subscription Peerings data source in the Terraform provider Redis Cloud.
---


# Data Source: rediscloud_subscription_peerings

The Subscription Peerings data source allows access to a list of VPC peerings for a particular subscription.

## Example Usage

The following example returns a list of all VPC peering within a Subscription.

```hcl-terraform
data "rediscloud_subscription_peerings" "example" {
  subscription_id = "1234"
}

output "rediscloud_subscription_peerings" {
  value = data.rediscloud_subscription_peerings.example.peerings
}
```

## Argument Reference

* `subscription_id` - (Required) ID of the subscription that the peerings belongs to
* `status` - (Optional) Current status of the peering - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`.

## Attributes Reference

* `peerings` A list of subscription peerings.

Each peering entry provides the following attributes

* `peering_id` - ID of the subscription peering
* `provider_name` - The name of the cloud provider. (either `AWS` or `GCP`)
* `status` Current status of the peering - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`.

**AWS ONLY:**

* `aws_account_id` AWS account id that the VPC to be peered lives in
* `vpc_id` Identifier of the VPC to be peered
* `vpc_cidr` CIDR range of the VPC to be peered
* `aws_peering_id` Identifier of the AWS cloud peering
* `region` AWS Region that the VPC to be peered lives in

**GCP ONLY:**
* `gcp_project_id` GCP project ID that the VPC to be peered lives in
* `gcp_network_name` The name of the network to be peered
* `gcp_redis_project_id` Identifier of the Redis Enterprise Cloud GCP project to be peered
* `gcp_redis_network_name` - The name of the Redis Enterprise Cloud network to be peered
* `gcp_peering_id` - Identifier of the cloud peering
