# Deprecation warnings on terraform apply

## TLDR

- `terraform apply` already warns for five deprecated attributes. Several other Cloud API deprecations are docs-only, so apply stays silent.
- Cloud API (`sm-cloud-api`) marks fields `@Deprecated` and `deprecated: true` in OpenAPI. It does not send an HTTP `Deprecation` or `Sunset` header. One field is rejected with HTTP 400.
- This change adds `provider/deprecation.Message` and uses it on every attribute that can warn safely. Apply still succeeds.
- Ticket: [RED-224042](https://redislabs.atlassian.net/browse/RED-224042). Status: Proposed. Assignee: Reouven Smadja. Drafted 2026-11-03. Not accepted.

## Architecture

```
Cloud API model                Terraform provider                 Customer
@Deprecated / OpenAPI    -->   schema Deprecated          -->     warning on
deprecated: true               or DeprecationMessage              terraform plan
                               or WhenNonEmpty validator          and apply
HTTP 400 (one field)     -->   still a warning today              apply succeeds
no Deprecation header    -->   provider must say it itself
```

There is no module-level `deprecated` block in Terraform. A public registry module cannot warn on apply by itself. The warning has to come from the provider schema.

## Problem

Customers keep using fields Cloud API has already deprecated. Some provider attributes warn. Others only say "deprecated" in the description or the docs, and `terraform apply` prints nothing.

## Goals

- One sentence for every deprecated attribute: name the attribute, name the replacement, and say it will be removed in the next major version of the provider.
- Warn on `terraform plan` and `terraform apply` while the attribute still works.
- Do not warn on a configuration that never set the attribute.

## Non-goals

- Removing any attribute. Removal stays a major version.
- Turning Cloud API's HTTP 400 into a provider error in this change.
- Adding replacement attributes that the provider does not have yet (`dataset_size_in_gb` on Essentials, a `replica` block on Pro databases).

## What warns today (provider `main`, 2.19.3)

These set SDKv2 `Deprecated`, so plan and apply warn and still succeed:

| Resource | Attribute | Replacement |
|---|---|---|
| `rediscloud_subscription` | `redis_version` | `redis_version` on the database |
| `rediscloud_subscription` | `creation_plan.average_item_size_in_bytes` | `ram_percentage` |
| `rediscloud_active_active_subscription` | `redis_version` | `redis_version` on the database |
| `rediscloud_subscription_database` | `average_item_size_in_bytes` | `ram_percentage` |
| `rediscloud_subscription_database` | `periodic_backup_path` | `remote_backup` |

`memory_limit_in_gb` is labeled deprecated in docs and in the attribute description on the Pro subscription creation plan, the Pro database, the Active-Active subscription creation plan, and the Active-Active database. Those schemas did not set `Deprecated` or `DeprecationMessage`, so apply stayed silent.

Separate warnings, not schema deprecation:

- Pro database create/update warns when `modules` is set on Redis 8.0+.
- Pro subscription creation warns when `average_item_size_in_bytes` or `ram_percentage` is set with `memory_storage = ram`.
- Docs call `throughput_measurement_by = number-of-shards` deprecated. The schema does not warn.

No resource and no module is deprecated as a whole. The Plugin Framework `DeprecationMessage` field was unused.

## What Cloud API does

Checked in `sm-cloud-api` data model and command processors. There is no runtime `Deprecation`, `Sunset`, or `Warning` response header. Deprecation is declared on the model and, in one case, enforced as an error.

| API field | How Cloud API signals it | Provider today |
|---|---|---|
| `redisVersion` on subscription create | `@Deprecated`. Description says use the database field. Still accepted. | Warns on Pro and Active-Active subscriptions. |
| `averageItemSizeInBytes` | `@Deprecated`. When Redis Flex RAM ratio is supported, create/update returns 400 `DATABASE_AVERAGE_ITEM_SIZE_IS_DEPRECATED`. | Warns. Apply can still fail later with the API 400. |
| `periodicBackupPath` on Pro database create/update | `@Deprecated`. Still accepted alone. 400 only when set together with `remoteBackup`. | Warns on `rediscloud_subscription_database`. |
| `periodicBackupPath` on Essentials | Not marked `@Deprecated`. | No warning. Leave it. |
| `memoryLimitInGb` on Pro and Essentials create | `@Deprecated`. Replacement is `datasetSizeInGb`. Still accepted. | Pro and Active-Active resources have `dataset_size_in_gb` but did not warn. Essentials has no `dataset_size_in_gb` attribute. |
| `clientSslCertificate` | `@Deprecated`. Replacement is `clientTlsCertificates`. | Both attributes exist. Pro did not warn. Active-Active did not warn. |
| `replicaOf` | `@Deprecated(since = "1.1.655", forRemoval = true)`. Replacement is `replica`. | Pro database still exposes `replica_of` and has no `replica` block. Essentials already uses `replica`. |
| `sslClientAuthentication` | `@Deprecated` in favor of `tlsClientAuthentication`. | Not a provider attribute. |
| `persistentStorageEncryption` | `@Deprecated` and hidden. | Removed from the provider in 0.3.0. |

OpenAPI `deprecated: true` shows up in the published schema and generated docs. A Terraform run never sees it unless the provider copies it into `Deprecated` / `DeprecationMessage`.

## Decisions

- **One helper, `deprecation.Message`.** Every warning detail is `` `<attribute>` is deprecated and will be removed in the next major version of the provider. Use <replacement> instead. `` Rejected: a different sentence per resource. Customers could not tell which warnings meant "this goes away".
- **Schema `Deprecated` or `DeprecationMessage` when the replacement attribute already exists and the attribute has no empty default.** Terraform prints that on validate, plan, and apply. This change sets it for `memory_limit_in_gb` on the Pro database, both subscription creation plans, and the Active-Active database.
- **`WhenNonEmpty` for `client_ssl_certificate` on the Pro database.** That attribute has `Default: ""`. Setting `Deprecated` on a defaulted attribute can warn every plan, including configurations that never set the certificate. The validator warns only when the string is non-empty. Active-Active `client_ssl_certificate` has no default, so it uses `DeprecationMessage`.
- **Do not warn yet when the replacement is missing from the provider.** Essentials `memory_limit_in_gb` and Pro `replica_of` stay silent. A warning that names an attribute the resource does not have is not a migration path.
- **Do not match the API 400 in this change.** `average_item_size_in_bytes` stays a warning. Promoting it to an error is a major-version decision.

## Implementation

- `provider/deprecation/message.go` — `Message`, `Warning`, `WhenNonEmpty`.
- Existing `Deprecated` strings on the five attributes above now call `Message`.
- New warnings: `memory_limit_in_gb` where `dataset_size_in_gb` exists; `client_ssl_certificate` on Pro (non-empty only) and Active-Active (schema deprecation message).
- Docs for those attributes use the same sentence.

## Testing

- `go test ./provider/deprecation/` covers the sentence and the empty-versus-set validator.
- Acceptance tests were not run. They need a Redis Cloud account, and this change does not alter create/update payloads.

## Operations / follow-ups

- Confirm whether `average_item_size_in_bytes` (`Default: 0`) and `periodic_backup_path` (`Default: ""`) already warn on every plan because of those defaults. If they do, switch them to `WhenNonEmpty` or an equivalent non-zero check.
- Add `dataset_size_in_gb` to Essentials, then warn on `memory_limit_in_gb`.
- Add the `replica` block next to `replica_of`, then warn.
- Decide whether `number-of-shards` and the Redis 8 `modules` block should use the same sentence.
- Decide whether the next major version should error, matching `DATABASE_AVERAGE_ITEM_SIZE_IS_DEPRECATED`, instead of warning.
- Data-source copies of these fields are computed. A computed attribute the user does not set does not warn. That is acceptable: the warning belongs on the resource the user configures.

## Open questions

- Should a Cloud API 400 stay a warning in Terraform until the next major version?
- Is "next major version" the removal commitment Reouven wants in the customer sentence, or should the sentence name a version?
