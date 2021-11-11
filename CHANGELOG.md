# Changelog
All notable changes to this project will be documented in this file.
See updating [Changelog example here](https://keepachangelog.com/en/1.0.0/)

## 0.2.5 (November 11, 2021)

### Changed

- Updates Terraform Plugin SDK to v2.8.0
- Updates additional dependencies contributing to build, (goreleaser-action 2.8.0)
- Updated README.md covering acceptance test execution

### Fixed

- Redis Cloud subscription update is failing due to missing payment method id [#149](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/149)
- Wrong syntax in example. [#153](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/153)

## 0.2.4 (July 24, 2021)

### Changed

- Updates additional dependencies contributing to build, (includes tfproviderlint v0.27.1)
- Updates location of compiled provider as well as go and terraform versions [#129](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/129)
- Updates Terraform Plugin SDK to v2.7.0
- Updates the subscription timeout value for update function to 30 minutes [#133](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/133)

### Fixed

- Fixed parsing of log-levels by removing date/time prefix [#132](https://github.com/RedisLabs/terraform-provider-rediscloud/pull/132)

## 0.2.3 (June 22, 2021)

### Changed

- Updates additional dependencies contributing to build

### Fixed

- replicaOf setting cannot be disabled from terraform [#121](https://github.com/RedisLabs/terraform-provider-rediscloud/issues/121)

## 0.2.2 (April 27, 2021)

### Changed

- Updates dependency terraform-plugin-sdk/v2 v2.6.1
- Updates dependency tfproviderlint v0.26.0
- Updates additional dependencies contributing to build

### Fixed
- Terraform wants to replace fresh imported peering [#102](https://github.com/RedisLabs/terraform-provider-rediscloud/issues/102)
- Need validation for length of the database name [#99](https://github.com/RedisLabs/terraform-provider-rediscloud/issues/99)
- Modules not included when creating DB on existing subscription in GCP [#98](https://github.com/RedisLabs/terraform-provider-rediscloud/issues/98)

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
- `network_deployment_cidr` is now required and to resolve issues with plan convergence after a successful Terraform apply
- `network_deployment_cidr` and `networking_vpc_id` were excluded from the hash calculation as 
- `networks` added to the `region` block in subscription resource and data source to allow reading all different CIDR and subnets in Multi-AZ subscription
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
