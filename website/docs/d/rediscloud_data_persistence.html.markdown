---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_data_persistence"
sidebar_current: "docs-rediscloud-data-persistence"
description: |-
  Data Persistence data source in the Terraform provider Redis Cloud.
---

# Data Source: rediscloud_data_persistence

The data persistence data source allows access to a list of supported data persistence options.  
Each option represents the rate at which a database will persist its data to storage.

## Example Usage

The following example returns all of the data persistence options available within your Redis Enterprise Cloud account.

```hcl-terraform
data "rediscloud_data_persistence" "example" {
}

output "data_persistence_options" {
  value = data.rediscloud_data_persistence.example.data_persistence
}
```

## Attributes Reference

* `data_persistence` A list of data persistence option that can be applied to subscription databases

Each data persistence option provides the following attributes

* `name` - The identifier of the data persistence option.

* `description` - A meaningful description of the data persistence option.
