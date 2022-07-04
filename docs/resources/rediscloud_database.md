---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_database"
description: |-
Database resource in the Terraform provider Redis Cloud.
---


The `module` block supports:

* `name` (Required) Name of the Redis Labs database module to enable

  Example:
  
  ```hcl
      module {
        name  = "RedisJSON"
      }
  
      module {
        name  = "RedisBloom"
      }
  ```

