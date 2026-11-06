# RED-191564 multiple_availability_zones plan behavior

Owner: Shay Berman. Date: 2026-11-06. Status: Proposed.
Ticket: [RED-191564](https://redislabs.atlassian.net/browse/RED-191564).

## TLDR

- Changing `multiple_availability_zones` on `rediscloud_subscription` now plans replacement.
- Replacement destroys the existing subscription and creates a new one.
- The Cloud API accepts `multipleAvailabilityZones` only on create. The old in-place update reported success and left Redis Cloud unchanged.

## Decision

Option A. The provider requires replacement when the value changes.

Option B is not available in the client this provider calls. `rediscloud-go-api` v0.53.0 puts `multipleAvailabilityZones` on `subscriptions.CreateRegion` and on the subscription read model `subscriptions.Region`. `subscriptions.UpdateSubscription` is only `name`, `paymentMethodId`, and `publicEndpointAccess`. The regions client can create or delete Active-Active regions. It has no multi-AZ update. `resourceRedisCloudProSubscriptionUpdateWithTimeout` never sends cloud regions.

The published docs already said a change forces a new resource. The schema did not. `multiple_availability_zones` had no `ForceNew`, and `cloudRegionsForceNewDiff` calls `ForceNew` on the region set. For a set of objects, that flag does not mark a nested value change as replacement, so Terraform planned an in-place update. Apply then ran the update function, which ignores the field, and reported one change.

`rediscloud_active_active_subscription` and `rediscloud_active_active_subscription_regions` do not have this attribute. No change there.

## Implementation

- `ForceNew` is set on `cloud_provider.region.multiple_availability_zones`.
- Registry docs state that replacement destroys the subscription.
- A subscription created by this provider still needs `creation_plan` in config before Terraform can create the replacement. That requirement already applies to every attribute that forces a new subscription.

## Testing

- Unit test `TestMultipleAvailabilityZonesChangePlansReplacement` plans `true` to `false` and `false` to `true`. Both diffs require replacement because of `multiple_availability_zones`.
- The same test changes `name` and checks that plan stays an in-place update.
- No acceptance test and no live Cloud API call.

## Operational impact

The next apply that toggles `multiple_availability_zones` deletes the subscription. Databases, networking, and data on that subscription are destroyed with it. Customers who only wanted the Redis Cloud UI to change must not apply that plan.

Rollback is reverting this provider change. A subscription already destroyed by an apply is not restored by rollback.
