# Changelog

All notable changes to this project will be documented in this file.
See updating [Changelog example here](https://keepachangelog.com/en/1.0.0/)

## 1.7.0 (29th April 2024)

### Changed

- Deprecated datasources: `rediscloud_subscription`, `rediscloud_database`
- Deprecated
  resources: `rediscloud_subscription`, `rediscloud_subscription_database`, `rediscloud_active_active_subscription_database`

### Added

- To replace the deprecated datasources: `rediscloud_flexible_subscription`, `rediscloud_flexible_database`
- To replace the deprecated
  resources: `rediscloud_flexible_subscription`, `rediscloud_flexible_database`, `rediscloud_active_active_database`
- Datasources for Active Active features: `rediscloud_active_active_subscription`, `rediscloud_active_active_database`
- Datasources for Essentials
  features: `rediscloud_essentials_plan`, `rediscloud_essentials_subscription`, `rediscloud_essentials_database`
- Resources for Essentials features: `rediscloud_essentials_subscription`, `rediscloud_essentials_database`
- `modules`/`global_modules` can be specified on Active Active Subscription/Database resources, enabling `RedisJSON`
- All Subscription resources include the `pricing` attribute
- All Database resources include `latest_backup_status` and `latest_import_status` attributes as appropriate

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
