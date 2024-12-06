---
page_title: "Migrating to version v2.X.X"
---

# Migrating to version v2.X.X

V2 introduced breaking changes with the removal the `latest_backup_status` and `latest_import_status` attributes from `rediscloud_active_active_subscription_database`,
`rediscloud_active_active_subscription_regions`, `rediscloud_subscription_database` and `rediscloud_essentials_database` resources.
With this change, plans will be much faster to compute and apply. If you rely on one those attributes, they are still available under the respective database data sources. 
See the examples below on how to migrate to use the data sources.

## Active-Active Subscription Database:

**Before:**
```hcl

resource "rediscloud_active_active_subscription_database" "database_resource" {
  # ...
}

output "latest_backup_status" {
  value = nonsensitive({
    for r in rediscloud_active_active_subscription_database.database_resource.override_region :
    r.name => r.latest_backup_status
  })
}

output "latest_import_status" {
  value = rediscloud_active_active_subscription_database.database_resource.latest_import_status
}
```

**After:**

```hcl
data "rediscloud_active_active_subscription_database" "database" {
  subscription_id = "..."
  name            = "..."
}

output "latest_backup_status" {
  value = {
    for r in data.rediscloud_active_active_subscription_database.database.latest_backup_statuses :
    r.region => r
  }
}

output "latest_import_status" {
  value = data.rediscloud_active_active_subscription_database.database.latest_import_status
}
```

## Active-Active Subscription Regions:

**Before:**
```hcl
resource "rediscloud_active_active_subscription_regions" "regions_resource" {
  # ...
}

output "latest_backup_status" {
  value = {
    for r in rediscloud_active_active_subscription_regions.regions_resource.region :
    r.region => r.database[*].latest_backup_status)
  }
}
```

**After:**

```hcl
data "rediscloud_active_active_subscription_database" "database" {
  subscription_id = "..."
  name            = "..."
}

output "latest_backup_status" {
  value = {
    for r in data.rediscloud_active_active_subscription_database.database.latest_backup_statuses :
    r.region => flatten(r.response[*].status)
  }
}
```

## Pro Subscriptions:

**Before:**
```hcl
resource "rediscloud_subscription_database" "database_resource" {
  # ...
}

output "latest_backup_status" {
  value = rediscloud_subscription_database.database_resource.latest_backup_status
}

output "latest_import_status" {
  value = rediscloud_subscription_database.database_resource.latest_import_status
}
```

**After:**

```hcl
data "rediscloud_database" "database" {
  subscription_id = "..."
  name            = "..."
}

output "latest_backup_status" {
  value = data.rediscloud_database.database.latest_backup_status
}

output "latest_import_status" {
  value = data.rediscloud_database.database.latest_import_status
}
```

## Essentials Subscriptions:

**Before:**
```hcl
resource "rediscloud_essentials_database" "database_resource" {
  # ...
}

output "latest_backup_status" {
  value = rediscloud_essentials_database.database_resource.latest_backup_status
}

output "latest_import_status" {
  value = rediscloud_essentials_database.database_resource.latest_import_status
}
```

**After:**

```hcl
data "rediscloud_essentials_database" "database" {
  subscription_id = "..."
  name            = "..."
}

output "latest_backup_status" {
  value = data.rediscloud_essentials_database.database.latest_backup_status
}

output "latest_import_status" {
  value = data.rediscloud_essentials_database.database.latest_import_status
}
```

