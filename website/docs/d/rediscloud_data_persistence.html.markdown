---
layout: "rediscloud"
page_title: "RedisCloud: rediscloud_data_persistence"
sidebar_current: "docs-rediscloud-data-persistence"
description: |-
  Data Persistence data source in the Terraform provider RedisCloud.
---

# rediscloud_data_persistence

Use this data source to get a list of supported data persistence options.  
A data persistence option represents the rate at which a database will persist data to storage.

## Example Usage

```hcl
data "rediscloud_data_persistence" "example" {
}
```


## Attributes Reference

* `data_persistence` A list of data persistence option that can be applied to subscription databases

Each data persistence option provides the following attributes

* `name` - The identifier of the data persistence option.

* `description` - A meaningful description of the data persistence option.
