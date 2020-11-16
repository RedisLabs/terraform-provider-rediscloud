---
layout: "rediscloud"
page_title: "RedisCloud: rediscloud_cloud_account"
sidebar_current: "docs-rediscloud-cloud-account"
description: |-
  Cloud Account data source in the Terraform provider RedisCloud.
---

# rediscloud_cloud_account

Use this data source to get the ID of a cloud account for use with the subscription resource.

## Example Usage

```hcl
data "rediscloud_cloud_account" "example" {
  exclude_internal_account = true
}
```

## Argument Reference

* `exclude_internal_account` - (Optional) Whether to exclude the Redis Labs internal cloud account.

* `provider_type` - (Optional) The cloud provider of the cloud account, (either `AWS` or `GCP`)

* `name` - (Optional) A meaningful name to identify the cloud account

## Attributes Reference

`id` is set to the ID of the found cloud account.

* `access_key_id` The access key ID associated with the cloud account
