---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_database"
description: |-
Database resource in the Terraform provider Redis Cloud.
---


The `modules` block supports:

* `name` (Required) Name of the Redis Labs database module to enable

  Example:
  
  ```hcl
    modules = [
        {
          "name": "RedisJSON"
        },
        {
          "name": "RedisBloom"
        }
    ]
  ```
