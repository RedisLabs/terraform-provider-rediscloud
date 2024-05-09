---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_regions"
description: |-
  Regions data source in the Redis Cloud Terraform provider.
---

# Data Source: rediscloud_regions

The Regions data source allows access to a list of supported cloud provider regions. These regions can be used with the subscription resource.

## Example Usage

The following example returns all of the supported regions available within your Redis Enterprise Cloud account.

```hcl-terraform
data "rediscloud_regions" "example" {
}

output "all_regions" {
  value = data.rediscloud_regions.example.regions
}
```

The following example show how the list of regions can be filtered by cloud provider, (`AWS` or `GCP`).

```hcl-terraform
data "rediscloud_regions" "example_aws" {
  provider_name = "AWS"
}

data "rediscloud_regions" "example_gcp" {
  provider_name = "GCP"
}
```

## Argument Reference

* `provider_name` - (Optional) The name of the cloud provider to filter returned regions, (accepted values are `AWS` or `GCP`).

## Attributes Reference

* `regions` A list of regions from either a single or multiple cloud providers.

Each region entry provides the following attributes

* `name` The identifier assigned by the cloud provider, (for example `eu-west-1` for `AWS`)

* `provider_name` The identifier of the owning cloud provider, (either `AWS` or `GCP`)
