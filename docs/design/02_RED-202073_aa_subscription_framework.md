# Active-Active subscription resource framework migration

Ticket: [RED-202073](https://redislabs.atlassian.net/browse/RED-202073)
Date: 2026-11-06
Status: proposed in the pull request for this change

## TLDR

- `rediscloud_active_active_subscription` moves from the SDKv2 provider onto the Plugin Framework provider.
- The protocol-v5 mux stays. Nested blocks stay blocks. Existing state is readable.
- The data source was already migrated and is unchanged.

## Architecture

```text
Terraform
   |
   +-- protocol v5 mux
          |
          +-- Plugin Framework
          |     rediscloud_active_active_subscription   (this change)
          |     rediscloud_active_active_subscription_database
          |     data.rediscloud_active_active_subscription
          |
          +-- SDKv2
                rediscloud_subscription (pro)
                regions, peering, and the other Active-Active resources
```

Create still posts one Active-Active subscription, waits until it is active or encryption-key pending, and writes the subscription id into state before returning a later wait error. Update still takes the subscription mutex. Delete still uses the same status machine as the SDKv2 resource.

## Decisions

- Copy the Active-Active database resource layout: schema, model, and CRUD in `provider/activeactive`, registered from the Framework provider and removed from the SDKv2 map.
- Nested SDKv2 blocks stay `ListNestedBlock` or `SetNestedBlock`. Pricing is computed but remains a block. The data source uses attributes because a deferred read must not look like an empty block. Sharing that attribute shape here would change resource state.
- `creation_plan`, its region set, and `customer_managed_key` get custom collection types because they are configurable and are not read back from the API. Maintenance reuses the existing custom list type. Pricing stays a plain list. Resource pricing is not sorted. The data source sorts. Sorting here would reshuffle subscriptions that already have pricing in state.
- SDK defaults (`payment_method`, `cloud_provider`, CMK enabled, grace period, public endpoint) are Optional and Computed with Framework defaults. Computed is required for a default and for replacing a planned value with state.
- `payment_method` and `creation_plan` ignore changes after create. `redis_version` replaces the subscription only when the new value is non-empty and different. Removing the deprecated attribute keeps the stored value. The grace-period upgrade suppression (`""` to `immediate` while CMK is off) runs in `ModifyPlan` and only for an existing resource.
- Maintenance and pricing use a plan modifier that copies state when configuration omits the block. `UseStateForUnknown` does not, because an omitted block is null.
- Absent API strings are stored as empty values, matching SDKv2 `d.Set`. A prior payment method id, AWS account id, CMK principal, or grace period is kept when the API omits it. Configuration-only fields (`creation_plan`, `redis_version`, customer-managed keys) are not overwritten by read.
- The region set uses normal Framework equality. The SDKv2 hash only used the region name, and the whole block is ignored after create, so the effective diff is the same. The unused `networking_vpc_id` branch is not ported.
- `read_operations_per_second` is described as read operations. The SDKv2 schema repeated the write-operations sentence.

## Implementation

- `provider/activeactive/resource_active_active_subscription.go` holds the schema, plan modifiers, and import.
- `resource_active_active_subscription_mapping.go` builds the create request and writes API subscription, maintenance, and pricing into the model.
- `resource_active_active_subscription_crud.go` implements create, read, update, and delete with the existing waiters, mutex, and database cleanup helper.
- The SDKv2 file `provider/resource_rediscloud_active_active_subscription.go` is removed. Pro subscription helpers it shared (`FlattenPricing`, `FlattenMaintenance`, `ReadPaymentMethodID`) stay, because the pro resource still calls them.
- Timeouts remain 30 minutes for create and update, and 10 minutes for read and delete.

## Testing

- Unit tests check schema against the model, custom-type round trips, plan modifiers, create-time `creation_plan` and region requirements, the create request body, pricing order and zero values, read preservation of config-only fields, CMK create, a failed wait that still stores the id, grace-period update bodies, name/tag/public-endpoint update, and the deletion status table.
- The pro subscription keeps the existing SDK unit tests for grace period and deletion. The Active-Active cases moved to the Framework resource.
- `go test ./provider/activeactive/`, `./provider/`, and `./provider/utils/` are the checks for this change. Acceptance tests and live Redis Cloud calls were not run.
- New resource code statement coverage is 82 percent. The non-CMK create path after the subscription is active is not unit tested: database cleanup sleeps for 30 seconds before it lists databases.

## Operations / follow-ups

- Run the existing Active-Active subscription acceptance tests on a real account before the 2.21.0 release. In particular, plan an already-created subscription whose configuration still contains `creation_plan`. That block cannot be marked Computed, and Terraform may reject a plan modifier that replaces it with state.
- A later change can sort resource pricing the same way as the data source, and can make the empty strings nullable. Either change would touch existing state, so it is out of scope here.
- ADR and BDR are not needed. This implements the framework-migration decision already used by the Active-Active database resource and the cloud account resource.
