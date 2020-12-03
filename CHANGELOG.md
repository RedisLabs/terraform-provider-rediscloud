# Changelog
All notable changes to this project will be documented in this file.
See updating [Changelog example here](https://keepachangelog.com/en/1.0.0/)

## 0.2.2 (Unreleased)

## 0.2.1 (December 3, 2020)

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
