---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_regions"
sidebar_current: "docs-rediscloud-regions"
description: |-
  Regions data source in the Terraform provider Redis Cloud.
---

# Data Source: rediscloud_regions

Use this data source to get a list of supported regions from supported cloud providers. These regions can be used with the subscription resource.

## Example Usage

```hcl
data "rediscloud_regions" "example" {
}
```

## Argument Reference

* `provider_name` - (Optional) The name of the cloud provider to filter returned regions, (accepted values are `AWS` or `GCP`).

## Attributes Reference

`regions` A list of regions from either a single or multiple cloud providers.

Each region entry provides the following attributes

* `name` The identifier assigned by the cloud provider, (for example `eu-west-1` for `AWS`)

* `provider_name` The identifier of the owning cloud provider, (either `AWS` or `GCP`)
