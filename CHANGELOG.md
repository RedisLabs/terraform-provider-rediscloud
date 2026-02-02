# Changelog

All notable changes to this project will be documented in this file.
See updating [Changelog example here](https://keepachangelog.com/en/1.0.0/)

# 2.10.4 (30th January 2026)

## Fixed
- `rediscloud_active_active_subscription_database`: Fixed "provider produced inconsistent result after apply" error when `override_global_password` is set to the same value as `global_password`.

# 2.10.3 (29th January 2026)

## Fixed
- `rediscloud_active_active_subscription`: Fixed CMK (Customer Managed Key) flow by properly handling the `encryption_key_pending` state during subscription creation and reads.

# 2.10.2 (27th January 2026)

## Changed
- Migrated `rediscloud_active_active_subscription_database` resource from Terraform SDK v2 to the Terraform Plugin Framework. This is an internal architectural change with no breaking changes to the resource schema or behaviour.
- Provider now uses muxing to serve resources from both SDK v2 and Plugin Framework simultaneously, enabling incremental migration of resources. 
- `rediscloud_private_link` and `rediscloud_active_active_private_link`: Delete now uses direct API endpoint instead of removing principals individually.
- `rediscloud_private_link`: Updated documentation to use availability zone IDs instead of names, and added database resource.

## Fixed
- `rediscloud_active_active_subscription_database`: Improved handling of `enable_default_user` inheritance between global and regional overrides.
- `rediscloud_active_active_subscription_database`: Fixed `global_data_persistence` to be computed, correctly reflecting API defaults when not explicitly configured.

# 2.10.1 (12th January 2026)

## Added
- Added missing documentation for `rediscloud_private_link_endpoint_script` and `rediscloud_active_active_private_link_endpoint_script` data sources.

## Fixed
- Fixed nil pointer dereference crashes in `rediscloud_database` and `rediscloud_active_active_subscription_database` data sources/resources when optional fields are missing from API response.
- `rediscloud_active_active_subscription`: Fixed spurious resource replacement when removing the deprecated `redis_version` field.
- `rediscloud_active_active_subscription_peering`: Fixed import failure when provider_name was not explicitly set.
- Fixed incorrect usage of data sources in example documentation.

## Changed
- Improved CI/CD pipeline with additional validation checks and security scanning.
- Improved test infrastructure and parallel resource cleanup.

# 2.10.0 (22nd December 2025)

## Added
- New `rediscloud_transit_gateway_route` resource to manage Transit Gateway routing (CIDRs) separately from the attachment for Pro subscriptions. This is the preferred way to manage CIDRs.
- New `rediscloud_active_active_transit_gateway_route` resource to manage Transit Gateway routing (CIDRs) separately from the attachment for Active-Active subscriptions. This is the preferred way to manage CIDRs.

## Fixed
- `rediscloud_transit_gateway_attachment`: Delete operation now handles "TGW_ATTACHMENT_DOES_NOT_EXIST" error gracefully, making destroy idempotent.
- `rediscloud_active_active_transit_gateway_attachment`: Delete operation now handles "TGW_ATTACHMENT_DOES_NOT_EXIST" error gracefully, making destroy idempotent.

# 2.9.0 (15th December 2025)

## Added
- New `rediscloud_transit_gateway_invitations` data source to retrieve pending Transit Gateway attachment invitations for Pro subscriptions.
- New `rediscloud_transit_gateway_invitation_acceptor` resource to accept Transit Gateway attachment invitations for Pro subscriptions.
- New `rediscloud_active_active_transit_gateway_invitations` data source to retrieve pending Transit Gateway attachment invitations for Active-Active subscriptions.
- New `rediscloud_active_active_transit_gateway_invitation_acceptor` resource to accept Transit Gateway attachment invitations for Active-Active subscriptions.

## Fixed
- `rediscloud_subscription`: Fixed "Provider produced inconsistent final plan" error when `networking_deployment_cidr` is not known until apply time (e.g., when the CIDR comes from another resource).
- `rediscloud_subscription` and `rediscloud_active_active_subscription_database`: Fixed default `source_ips` values to correctly use RFC1918 private ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) when public endpoint access is
  disabled.
- `rediscloud_subscription` and `rediscloud_active_active_subscription_database`: Fixed `source_ips` values now correctly migrate when toggling `enable_public_endpoint`.
- `rediscloud_pro_database`: Fixed import behaviour for `redis_version` field to prevent post-import drift.
- Improved reliability of subscription and database state transitions with additional wait conditions.

# 2.8.0 (10th November 2025)

## Added
- Added support for database version for Essentials databases.
- Added `aws_account_id` attribute to Pro and Active-Active subscription resources and data sources.
- Added `region_id` to the attribute reference documentation for `rediscloud_active_active_subscription_regions` data source.
- Added `region_id` attribute to `rediscloud_regions` data source.
- Added `db_id` to the attribute reference documentation for `rediscloud_database` data source.

## Fixed
- Spurious diffs for `customer_managed_key_deletion_grace_period` are now suppressed when upgrading the provider.

# 2.7.4 (7th November 2025)

## Changed
- Reverted global/regional override rework from v2.7.3 due to regressions. Active-Active database global configuration behavior has been restored to v2.7.2 state. Transit Gateway improvements from v2.7.3 have been preserved.
- `rediscloud_active_active_subscription_database`: Both `global_enable_default_user` and the region-level `enable_default_user` (in `override_region` blocks) now default to `true`. To disable the default user in a specific region, you must explicitly set `enable_default_user = false` in that region's `override_region` block.

# 2.7.3 (6th November 2025)

## Changed
- Reworked the entire interaction between global/regional overrides and how they read config and state. This should fix many existing subtle state drift bugs.

## Fixed
- `rediscloud_active_active_subscription_database`: Fixed multiple issues concerning regional `enable_default_user` and `global_enable_default_user` to do with drift or incorrectly not detecting changes.
- The default for `global_enable_default_user` was omitted, it is now set to true.
- `rediscloud_active_active_transit_gateway_attachment`: Fixed parameter order bug in update operation.

## Testing
- Added acceptance tests covering `enable_default_user` inheritance and override scenarios
- Added acceptance test for `rediscloud_active_active_transit_gateway_attachment` resource lifecycle (Create/Read/Update/Delete/Import)

# 2.7.2 (3rd November 2025)

## Fixed
- rediscloud_active_active_subscription_database: Fixed state management for global configuration fields (`global_enable_default_user`, `global_data_persistence`, `global_password`). These fields are now read directly from the API response instead of being preserved from configuration, eliminating state drift issues and ensuring accurate change detection.
- `rediscloud_active_active_subscription_database`: Fixed issue where setting `global_enable_default_user = false` was silently ignored during updates. The provider now correctly handles boolean `false` values.
- `rediscloud_private_link_endpoint_script` and `rediscloud_active_active_private_link_endpoint_script` datasources: Updated to support changes in the underlying API structure for endpoint scripts.


# 2.7.1 (27th October 2025)

## Fixed
- rediscloud_subscription_database: The query_performance_factor attribute can now be updated in-place without recreating the database. Previously, any changes to this attribute would force resource replacement.
- When you remove the deprecated field `redis_version` field in subscriptions it should not attempt to force renew the resource.
- rediscloud_subscription_database (Redis 8.0+): Fixed drift detection issues where explicitly configured modules would incorrectly show as changes requiring resource replacement after upgrading to Redis 8.0 or higher. Modules are bundled
  by default in Redis 8.0+, so configuration differences are now properly suppressed.
- rediscloud_subscription_database (Redis 8.0+): The warning for modules has been made more prominent.
- Support for a new pending status for subscription and database updates.
- Test Suite: Fixed incorrect file path references in acceptance tests.

# 2.7.0 (22nd October 2025)

## Added:
- Add auto_minor_version_upgrade field to Pro and Active-Active database resources (default: true) to allow users to control automatic minor version upgrades. This will NOT affect existing databases.

## Changed:
- Change Redis 8.0 modules validation from hard error to warning since modules are bundled by default in Redis 8+.

## Fixed:
- Fix test error message patterns to match updated API error format.
- Fix Redis 8 upgrade test expectation (dataset_size_in_gb: 3→1).


# 2.6.0 (17th October 2025)

## Added:
- Support for disabling public endpoints on databases. When public endpoints are disabled, database connections are restricted to private networks only (via VPC peering, PrivateLink, or Private Service Connect).
- `source_ips` attribute added to `rediscloud_database` data source.
- `global_source_ips` attribute added to `rediscloud_active_active_subscription_database` data source.

## Fixed:
- The default value for `enable_default_user` on each region for active-active subscriptions made the global default effectively redundant. The default has been removed meaning that the global default should work correctly now.

# 2.5.0 (13th October 2025)

## Added:
- Support for Redis 8 databases and upgrading. Redis 8 does not have modules so the provider should handle these gracefully. 
- Support for `query_performance_factor` on Redis 8.0 - Updated validation logic to allow QPF on Redis 8.0+ databases since RediSearch is bundled by default.

## Fixed:
- Fix subscription state handling after Redis version upgrades - Added wait for subscription to become active after upgrading Redis versions to prevent "SUBSCRIPTION_NOT_ACTIVE" errors during subsequent operations.
- Added full compatibility with Redis 8.2 and later.
- Older provider versions (<2.5.0) will fail during 8.2 database creation.

## Changed:

- Refactor inline pro Terraform configs to external files.
- Optimize test execution time by downsizing some configs

# 2.4.5 (9th October 2025)

## Added:
- Support for the global_enable_default_user attribute to Active-Active database resources, allowing users to control whether the default Redis user is enabled across all regions.

# 2.4.4 (3rd October 2025)

## Changed:
- Fixed AA endpoint script calling wrong method
- Fixed connection flattening not aligning to schema

# 2.4.3 (1st October 2025)

## Added:
- PrivateLink pro subscription tests

## Changed:
- Fixed recursive delete loop on PrivateLink deletes
- Fixed import ID mismatches and subscription IDs to be more idiomatic
- Updated API SDK to fix some API schema issues for PrivateLink
- Fixed schema mismatches that could cause issues
- Fixed description on some of the PrivateLink schema that was incorrect

# 2.4.2 (29th September 2025)

### Added

- Gives users the option to disable a default user in active-active regions by setting the `enable_default_user` flag to false.

### Fixed

- Expanded PrivateLink documentation and fixed errata

# 2.4.1 (24th September 2025)

### Added

- AWS PrivateLink support for Active Active subscriptions.
- New resource: `rediscloud_active_active_private_link`
- New data source: `rediscloud_active_active_private_link`
- New data source: `rediscloud_active_active_private_link_endpoint_script`

### Changed:

- `rediscloud_active_active_subscription_regions` now supports the property `region_id`

# 2.4.0 (19th September 2025)

### Added

- AWS PrivateLink support for Pro Subscriptions.
- New resource: `rediscloud_private_link` which allows users to manage peering between Redis Subscriptions and AWS resources.
- New data source: `rediscloud_private_link` 
- New data source: `rediscloud_private_link_endpoint_script`

# 2.3.0 (19th August 2025)

### Added

- Redis Database version support on create. Specify a version on create to create a DB of that version.
- Database upgrade path. If you specify a different version to something already specified, the provider will upgrade your database to the new version. Will fail on downgrade.
- Redis AA Database version support on create. No upgrade path yet - if you change your `redis_version` it will force a new resource.
- Updated data sources for pro and active active databases to also support `redis_version`.

### Changed
- Updating multiple dependencies.
- Deprecate subscription version support. If you use `redis_version` on your pro subscription a warning will come up. This will be removed entirely on a major update.

# 2.2.0 (1st August 2025)

### Added

- Customer Managed Key support for active-active and pro subscriptions. Only supports redis internal GCP cloud subscriptions. CMKs are externally provided by a customer-supplied GCP account and are managed externally by the user.

# 2.1.5 (1st July 2025)

### Added

- Feature: Support Marketplace as a payment method for Essentials subscription
- Feature: Add TLS certificate to databases’ data sources

### Fixed:

- Unexpected state `dynamic-endpoints-creation-pending'
- Can not disable default user on essentials db

# 2.1.4 (22nd May 2025)

### Added

- Documentation for `rediscloud_active_active_subscription_regions` added.
- Schema documentation amended to match documentation above.

# 2.1.3 (21st May 2025)

### Added

- New datasource `rediscloud_active_active_subscription_regions` added.

# 2.1.2 (19th May 2025)

### Added

- Query Performance Factor now implemented for resources `rediscloud_subscription` and `rediscloud_subscription_database`
- Reducing the number of attachments for Private Service Connect from 40 down to 1, in tests and documentation

# 2.1.1 (6th Feb 2025)

### Added

- Documentation related to using the [Redis Cloud Private Service Connect Module](https://github.com/RedisLabs/terraform-rediscloud-private-service-connect)
  to simplify the Terraform configuration.

# 2.1.0 (6th Feb 2025)

### Added

- Added resources for provisioning Private Service Connect in GCP on Active-Active and Pro subscriptions.
`rediscloud_private_service_connect`, `rediscloud_private_service_connect_endpoint`, `rediscloud_private_service_connect_endpoint_accepter`,
`rediscloud_active_active_private_service_connect`, `rediscloud_active_active_private_service_connect_endpoint` and
`rediscloud_active_active_private_service_connect_endpoint_accepter` as well as the respective data sources
`rediscloud_private_service_connect`, `rediscloud_private_service_connect_endpoints`, `rediscloud_active_active_private_service_connect`
and `rediscloud_active_active_private_service_connect_endpoints`

### Changed

- Upgraded the provider to use `v0.22.0` of the [rediscloud-go-api](https://github.com/RedisLabs/rediscloud-go-api) SDK.

# 2.0.0 (5th Dec 2024)

### Changed

- Upgraded the provider to use `v0.21.0` of the [rediscloud-go-api](https://github.com/RedisLabs/rediscloud-go-api) which handles API rate limits gracefully.

### Removed

- `latest_backup_status` and `latest_import_status` from `rediscloud_active_active_subscription_database`,
  `rediscloud_active_active_subscription_regions`, `rediscloud_subscription_database` and `rediscloud_essentials_database`.
  Users should use the equivalent data sources instead.

# 1.9.0 (9th Oct 2024)

### Added

- Tags as key/value pairs on Pro and ActiveActive databases
- The facility for users to upgrade from memorySizeInGb to datasetSizeInGb
(please note that changing back may have unexpected results and is not supported)

# 1.8.1 (3rd Sept 2024)

### Removed

- Validation preventing measurement of throughput by ops/sec with the Redisearch module present

### Fixed

- Slight performance improvement

# 1.8.0 (12th August 2024)

### Added

- Maintenance Windows properties on Pro and ActiveActive Subscriptions
- Transit Gateway Datasources and TGw Attachment Resources for Pro and ActiveActive subscriptions
- TLS certificates for Pro and ActiveActive Databases
- An optional Subscription id argument in the Essentials Plan Datasource

### Removed

- Provider validation on Alert names in Essentials Databases

### Fixed

- Updating ACL Rules
- Fixed parallelism when creating over 2 Databases under one Subscription

### Changed

- Documentation improvements
- Provider now reacts to externally changed SSL/TLS credentials and notifies the user via `plan`
- Fixed a bug related to providing empty strings in a list

## 1.7.0 (11th June 2024)

### Fixed

- Datasources `rediscloud_subscription` and `rediscloud_databases` are for Pro plans only

### Added

- Resources for Essentials plans: `rediscloud_essentials_subscription`, `rediscloud_essentials_database`
- Datasources for Active-Active
  deployments: `rediscloud_active_active_subscription`, `rediscloud_active_active_subscription_database`
- Datasources for Essentials
  plans: `rediscloud_essentials_plan`, `rediscloud_essentials_subscription`, `rediscloud_essentials_database`
- `modules`/`global_modules` can be specified on Active-Active Subscription/Database resources, enabling `RedisJSON`
- All Subscription resources include the `pricing` attribute
- All Database resources include `latest_backup_status` and `latest_import_status` attributes as appropriate
- The `redis_version` attribute for `rediscloud_subscription` now supports numeric versions as input

## 1.6.0 (12 April 2024)

### Fixed

- using the `rediscloud_database` datasource no longer crashes when pointed to an ActiveActive database but offers
  limited data. A specific datasource type will be coming soon.

### Changed

- A subscription's `payment_method` can no longer be updated. It is ignored after resource creation (as
  with `creation_plan`).
  This means if it has been changed behind the scenes, reapplying the same Terraform configuration should not change
  anything.

## 1.5.0 (24 November 2023)

### Added

- a `resp_version` property on active-active databases
- a `resp_version` property on active-active regions

## 1.4.0 (21 November 2023)

### Added

- a `resp_version` property on single-region databases
- an `enable_default_user` property on single-region databases
- a `redis_version` property on both single-region and active-active subscriptions

## 1.3.3 (10 November 2023)

### Fixed

- Improved waiting/timeout behavior, including making use of the `status` property on ACL Users.
- Measuring throughput by `operations-per-second` is incompatible with the `RediSearch` module.
- Slight documentation changes.
- Alerts can be removed from databases as expected.

### Removed

- The `REDISCLOUD_SUBSCRIPTION_TIMEOUT` environment variable is gone. Subscription creation times out after the user's
  setting (or 30 minutes by default). Note there is a 6-hour cap, regardless of the user's setting.

## 1.3.2 (9 October 2023)

### Added

- Added a new environment variable `REDISCLOUD_SUBSCRIPTION_TIMEOUT` to allow
  configuring timeouts for subscriptions at the provider level.
  This is a **TEMPORARY** solution and will be deleted in the next releases.

## 1.3.1 (10 August 2023)

### Fixed

- Documentation fixes

## 1.3.0 (7 August 2023)

### Added

- Added ACL resources and data sources (users, roles, rules)

## 1.2.0 (9 June 2023)

### Added

- Add support for using a custom port number in normal or active/active databases
- Add support for configuring backups for normal or active/active databases
- Add support for peering normal or active/active subscriptions with AWS VPCs that use multiple CIDR ranges

### Fixed

- Documentation fixes
- Make CI runs stream test output rather than batching it up at the end

### Changed

- `rediscloud_subscription.preferred_availability_zones` changed to optional
- `rediscloud_subscription.modules` changed to optional
- `rediscloud_subscription_database.protocol` changed to default to `redis`
- Mark `rediscloud_subscription_database.periodic_backup_path` as deprecated - use `remote_backup` instead.
- Emit a warning if `average_item_size_in_bytes` has been specified when `memory_storage` is set to `ram` as this
  attribute is only applicable with `ram-and-flash` storage.

## 1.1.1 (6 March 2023)

### Fixed

- Documentation fixes

## 1.1.0 (6 March 2023)

### Added

- Added support for active/active databases

### Fixed

- Documentation fixes

## 1.0.3 (1 February 2023)

### Fixed

- Documentation fixes

## 1.0.2 (16 January 2023)

### Fixed

- Documentation fixes

## 1.0.1 (12 September 2022)

### Changed

- Changed the `average_item_size_in_bytes` attribute of the creation_plan block to send a “null” value to the API if
  omitted.

### Fixed

- Various documentation fixes
- Fixed an issue where the `source_ips` and `enable_tls` attributes were not being provisioned correctly on
  the `rediscloud_subscription_database` resource

## 1.0.0 (30 August 2022)

### Added

- Added the creation_plan block in the subscription resource schema.
- Added a new resource type: `rediscloud_subscription_database`.
- Added the migration guide to help users with migrating their old Terraform configuration files to v1.0.0.
- Multi-modules: Added the "modules" attribute into the creation_plan block
  and the database resource schema

### Changed

- Updates to dependencies and CI related actions

### Removed

- Removed the database block from the subscription resource schema.

## 0.3.0 (May 24 2022)

### Added

- Added support for the DataEviction attribute in the _database_ data source and _subscription_ resource
- Added paymentMethod field to Subscription resource

### Removed

- Removed a deprecated attribute: persistent_storage_encryption

### Changed

- Updates rediscloud-go-api to v0.1.7: removed the persistent_storage_encryption attribute from the API calls
- Adds region attribute to Peering resource (for Read method) and data resource
- Patch a vulnerability: CVE-2022-29810 by upgrading go-getter v1.5.3 -> v1.5.11
- Fix timing on large subscription: reduce PUT requests

## 0.2.9 (March 28 2022)

### Changed

- Updates additional dependencies contributing to build, (goreleaser-action 2.8.1)
- Updates Terraform Plugin SDK to v2.10.1
- Updates rediscloud-go-api dependency to v0.1.6 use correct content-type with API

## 0.2.8 (December 14 2021)

### Changed

- Updates Subscription database to enable TLS
- Updates Terraform Plugin SDK to v2.10.0

## 0.2.7 (November 26 2021)

### Fixed

- Adjusts goreleaser configuration with go 1.17

## 0.2.6 (November 26 2021)

### Changed

- Updates Terraform Plugin SDK to v2.9.0
- Updates Go version [#162](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/162)
- Updates additional dependencies contributing to build

## 0.2.5 (November 11, 2021)

### Changed

- Updates Terraform Plugin SDK to v2.8.0
- Updates additional dependencies contributing to build, (goreleaser-action 2.8.0)
- Updated README.md covering acceptance test execution

### Fixed

- Redis Cloud subscription update is failing due to missing payment method
  id [#149](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/149)
- Wrong syntax in example. [#153](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/153)

## 0.2.4 (July 24, 2021)

### Changed

- Updates additional dependencies contributing to build, (includes tfproviderlint v0.27.1)
- Updates location of compiled provider as well as go and terraform
  versions [#129](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/129)
- Updates Terraform Plugin SDK to v2.7.0
- Updates the subscription timeout value for update function to 30
  minutes [#133](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/133)

### Fixed

- Fixed parsing of log-levels by removing date/time
  prefix [#132](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/132)

## 0.2.3 (June 22, 2021)

### Changed

- Updates additional dependencies contributing to build

### Fixed

- replicaOf setting cannot be disabled from
  terraform [#121](https://github.com/RedisLabs/terraform-provider-rediscloud/issues/121)

## 0.2.2 (April 27, 2021)

### Changed

- Updates dependency terraform-plugin-sdk/v2 v2.6.1
- Updates dependency tfproviderlint v0.26.0
- Updates additional dependencies contributing to build

### Fixed

- Terraform wants to replace fresh imported
  peering [#102](https://github.com/RedisLabs/terraform-provider-rediscloud/issues/102)
- Need validation for length of the database
  name [#99](https://github.com/RedisLabs/terraform-provider-rediscloud/issues/99)
- Modules not included when creating DB on existing subscription in
  GCP [#98](https://github.com/RedisLabs/terraform-provider-rediscloud/issues/98)

## 0.2.1 (December 7, 2020)

### Changed

- Updates to expose additional vpc peering attributes for GCP and AWS
- Updates docs to include new attributes and expands examples to include output value for peering commands
- Updates rediscloud-go-api to release v0.1.3

## 0.2.0 (November 30, 2020)

### Added

- Support for GCP Subscription Peering
- datasource_rediscloud_subscription_peerings to retrieve the subscription peering details

### Changed

- Website documentation and HCL examples to correct spelling and update content
- Changelog to record released content
- `network_deployment_cidr` is now required and to resolve issues with plan convergence after a successful Terraform
  apply
- `network_deployment_cidr` and `networking_vpc_id` were excluded from the hash calculation as
- `networks` added to the `region` block in subscription resource and data source to allow reading all different CIDR
  and subnets in Multi-AZ subscription
- Fixed issues when creating a subscription without a payment method

### Removed

- `network_deployment_subnet` was moved to the `networks` block in subscription resource and data source

## 0.1.1 (November 24, 2020)

### Added

- Released through `registry.terraform.io` RedisLabs/rediscloud

## 0.1.0 (November 24, 2020)

### Added

- resource_rediscloud_cloud_account to allow management of cloud account credentials
- resource_rediscloud_subscription to allow management of subscriptions and databases
- resource_rediscloud_subscription_peering to allow the management of AWS VPC Peering requests
- datasource_rediscloud_cloud_account to retrieve the details of configured cloud accounts
- datasource_rediscloud_data_persistence to view the available data persistence options
- datasource_rediscloud_database to view the details of an existing database
- datasource_rediscloud_database_modules to view the available database modules
- datasource_rediscloud_payment_method retrieve the details of configured payment methods
- datasource_rediscloud_regions to view a list of supported cloud provider regions
- datasource_rediscloud_subscription to view the details of an existing subscription
- Website documentation and acceptance tests

### Changed

- Migrated website documentation to new structure and format
- README to cover local builds, developing and testing the Provider
