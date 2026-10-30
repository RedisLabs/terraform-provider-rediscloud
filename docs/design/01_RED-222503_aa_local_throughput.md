# Active-Active local throughput on the database resource

## TLDR

- `override_region` on `rediscloud_active_active_subscription_database` can set local read and write ops per second.
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
- Both attributes are required together through `int64validator.AlsoRequires`.

## Implementation

- Create, update, and read in the Active-Active database resource set the throughput on the existing request builders.
- Each value must be at least 1.

## Testing

- A unit test checks that an unrelated region update omits throughput when the attributes are unset, and that refresh leaves them null.
- `TestAccResourceRedisCloudActiveActiveDatabase_LocalThroughput` creates a real Active-Active database, checks `GetActiveActive`, then changes the numbers in place. A region that omits the attributes stays at 1000/1000.
- The pull-request smoke suite and the main database suite both run that acceptance test.

## Operations / follow-ups

- Do not set the same database's throughput on both this resource and `rediscloud_active_active_subscription_regions`.
