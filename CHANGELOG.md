# Changelog
All notable changes to this project will be documented in this file.
See updating [Changelog example here](https://keepachangelog.com/en/1.0.0/)

## 0.1.2 (Unreleased)

### Added

### Changed

- Website documentation and HCL examples to correct spelling and update content
- Changelog to record released content

### Removed


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
