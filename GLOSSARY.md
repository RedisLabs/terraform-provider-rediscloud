# Glossary

Repository terms for the Redis Cloud Terraform provider. Meanings only.

## Creation plan

`creation_plan` is the on-create block on a Redis Cloud subscription resource. SM cannot create a subscription with no database, and the provider still keeps the subscription and the database as separate resources. The block creates a temporary database so the subscription can come up, then deletes it. Later updates and import ignore the block.

An empty subscription means a subscription with no database at create time. Support for that was discussed and dropped. The provider has to keep `creation_plan`.

Aliases: creation plan, creation plan block.

Source: Teodor Stoykov, meeting with Shay Berman, 2026-10-29, recorded in `terraform-provider-rediscloud-private` at `docs/knowledge-transfer/sessions/2026-10-29-teodor-maintainer/README.md`. The same create-then-delete flow is described on the public `rediscloud_subscription` resource page. Recorded 2026-10-29.

## Crossplane provider

The Redis Cloud Crossplane provider is [RedisLabs/crossplane-provider-rediscloud](https://github.com/RedisLabs/crossplane-provider-rediscloud). It is generated from this Terraform provider with Upjet. A daily job in that repository opens a pull request when this provider publishes a newer release. The job does not merge and does not publish a Crossplane package.

Aliases: Crossplane, `crossplane-provider-rediscloud`.

Source: [docs/design/downstream-provider-releases.md](docs/design/downstream-provider-releases.md). Recorded 2026-11-05.

## Private provisioning repo

`redislabsdev/terraform-provider-rediscloud-private` holds the account-lifecycle automation that acceptance tests of this public provider dispatch. It creates and deletes throwaway Redis Cloud test accounts and returns Cloud API credentials encrypted to the caller. The public provider never contains that logic.

Aliases: `terraform-provider-rediscloud-private`.

Source: that repo's README ("Why this repo exists") and this repo's `.github/workflows/testacc.yml`. Recorded 2026-10-29.

## Pulumi provider

The Redis Cloud Pulumi provider is the Terraform bridge in [RedisLabs/pulumi-rediscloud](https://github.com/RedisLabs/pulumi-rediscloud), published on the Pulumi registry as package `rediscloud`. It is not a service in this repository and it is not deployed as a cluster workload.

The package customers install is still **v1.3.5** (20 December 2023), built from this provider **v1.3.1**. Unpublished `main` pins a later release of this provider. A daily job in that repository opens a pull request when this provider publishes a newer release. The job does not merge and does not publish a Pulumi package. The process is [docs/design/downstream-provider-releases.md](docs/design/downstream-provider-releases.md).

Customers also reach Cloud API with Pulumi's generic Terraform bridge. Those calls show up as user agent `Terraform/pulumi-terraform-bridge` plus `terraform-provider-rediscloud/<version>`, including versions newer than v1.3.1. That path has no separate Redis repository.

Aliases: Pulumi, `pulumi-rediscloud`.

Source: [RedisLabs/pulumi-rediscloud](https://github.com/RedisLabs/pulumi-rediscloud) `provider/go.mod` and `provider/resources.go`; Pulumi registry package `rediscloud` v1.3.5; [RED-159267](https://redislabs.atlassian.net/browse/RED-159267). Updated 2026-11-05 from the downstream release design.
