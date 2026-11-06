# Custom hashing policy on database create

## TLDR

- `rediscloud_subscription_database` (the pro database resource) sends `shardingType: custom-regex-rules` on create when `hashing_policy` is set.
- Databases that omit `hashing_policy` do not send a sharding type, so redis-oss hashing stays the default.
- Adding or removing `hashing_policy` replaces the database. Editing the regex list updates in place, and an update that does not change it does not send `regexRules`.

## Status

- Ticket: [RED-208367](https://redislabs.atlassian.net/browse/RED-208367)
- Date: 2026-11-06
- Owner: Teodor Stoykov (assignee). Implemented from Maor Ohana's diagnosis, checked against this provider, `rediscloud-go-api` v0.53.0, and `sm-cloud-api`.

## Problem

Customers who set `hashing_policy` on `rediscloud_subscription_database` get `400 DATABASE_SHARDING_TYPE_IS_IMMUTABLE` ("You cannot change the hashing policy"). The same Terraform resource is what the provider registers for the pro database. It worked before redis-oss hashing became the default.

`hashing_policy` is a list of regex rules. Those rules are valid for the custom sharding type (`custom-regex-rules`), not for `redis-oss-hashing`. The provider created the database without a sharding type, then its follow-up update sent the regex rules. Cloud API treats that update as a hashing-policy change on a database that is already redis-oss, and rejects it.

## Decisions

- Do not add a Terraform `sharding_type` attribute. `rediscloud-go-api` v0.53.0 `CreateDatabase` has no sharding type field, and customers already express the custom policy through `hashing_policy`.
- When `hashing_policy` is non-empty, the create request uses `custom-regex-rules`. When it is omitted, the request leaves `shardingType` unset so the account default remains.
- The Go client cannot set the field, so the provider HTTP transport adds it to `POST /subscriptions/{id}/databases` only when that create asked for it. A body that already contains `shardingType` is left alone.
- `UpdateDatabase` in the Go client has `regexRules` and no sharding type. Cloud API `validateUpdateShardingType` allows a regex-rule change once the database is already `custom-regex-rules`, and rejects rules sent for `redis-oss-hashing`. The provider therefore sends `regexRules` only when `hashing_policy` is changing to a non-empty list. Adding or removing the attribute forces a new database, because that is the immutable hashing-policy change. Editing the list does not.
- Active-Active databases have no `hashing_policy` attribute, so there is no create path to change. Essentials uses `regex_rules` on the fixed-database create request, which already derives the sharding type. That resource is out of this fix.
- No new ADR. This follows the existing Cloud API contract. No BDR.

## Implementation

- `provider/pro` sets the create context when `hashing_policy` is configured, and builds the update `regexRules` from a real change of that attribute.
- `provider/client` transport writes `shardingType` onto that one POST. The SDK provider and the unit-test API client both install the transport.
- Read behaviour is unchanged. The two built-in default regex rules are still stored as an empty `hashing_policy`.

## Testing

- Create with `hashing_policy` sends `custom-regex-rules` and then `regexRules`.
- Create without `hashing_policy` sends neither.
- An update that does not change `hashing_policy` omits `regexRules`.
- The replacement helper covers add, remove, edit, and create.
- No live API calls.

## Operations

- Accounts still need the custom hashing policy feature to accept `custom-regex-rules`. If it is off, create fails with sharding type not supported. That is separate from the immutable-change error.
- Rollback is reverting this provider change. Databases already created with a custom hashing policy stay as they are.

## Documentation

Design: `docs/design/04_RED-208367_hashing_policy.md`. ADR not needed. BDR not needed.
