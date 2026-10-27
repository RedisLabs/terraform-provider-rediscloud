# Active-Active local throughput on the database resource

## TLDR

- `override_region` on `rediscloud_active_active_subscription_database` can set local read and write ops per second.
- Ticket: [RED-222503](https://redislabs.atlassian.net/browse/RED-222503)
- The Cloud API already accepts the values. Regions that omit the attributes stay at 1000/1000.

## Architecture

```text
override_region.local_*_operations_per_second
        |
        +-- create --> POST /subscriptions/{id}/databases
        |              localThroughputMeasurement[]
        |
        +-- update --> PUT /subscriptions/{id}/databases/{dbId}/regions
                       regions[].localThroughputMeasurement
```

## Decisions

- The two attributes live on `override_region` because throughput is per region and that block already exists per region.
- Unset attributes are not sent on update, so this resource does not overwrite throughput managed by `rediscloud_active_active_subscription_regions`.
- Read stores the API values only when the user set the attributes, so existing configurations do not plan a diff.

## Implementation

- `provider/activeactive/local_throughput.go` builds the create list, the update field, and the read values.
- Schema, model, create, update, and read in the Active-Active database resource.
- Both attributes are required together, and each must be at least 1.

## Testing

- Unit tests cover configured values, the 1000/1000 default, omitted update fields, and read-back.
- `TestAccResourceRedisCloudActiveActiveDatabase_LocalThroughput` creates a real Active-Active database, checks `GetActiveActive`, then changes the numbers in place. A region that omits the attributes stays at 1000/1000.

## Operations / follow-ups

- Do not set the same database's throughput on both this resource and `rediscloud_active_active_subscription_regions`.
