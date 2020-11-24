---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_cloud_account"
sidebar_current: "docs-rediscloud-cloud-account"
description: |-
  Cloud Account data source in the Terraform provider Redis Cloud.
---

# Data Source: rediscloud_cloud_account

The Cloud Account data source allows access to the ID of a Cloud Account configuration.  This ID can be 
used when creating Subscription resources. 

## Example Usage

The following example excludes the Redis Labs internal cloud account and returns only AWS related accounts.
This example assumes there is only a single AWS cloud account defined.

```hcl-terraform
data "rediscloud_cloud_account" "example" {
  exclude_internal_account = true
  provider_type = "AWS"
}
```

If there is more than one AWS cloud account then the name attribute can be used to further filter the ID returned.
This example looks for a cloud account named `test` and returns the cloud account ID and access key ID. 

```hcl
data "rediscloud_cloud_account" "example" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "test"
}

output "cloud_account_id" {
  value = data.rediscloud_cloud_account.example.id
}

output "cloud_account_access_key_id" {
  value = data.rediscloud_cloud_account.example.access_key_id
}

```

## Argument Reference

* `exclude_internal_account` - (Optional) Whether to exclude the Redis Labs internal cloud account.

* `provider_type` - (Optional) The cloud provider of the cloud account, (either `AWS` or `GCP`)

* `name` - (Optional) A meaningful name to identify the cloud account

## Attributes Reference

`id` is set to the ID of the found cloud account.

* `access_key_id` The access key ID associated with the cloud account
