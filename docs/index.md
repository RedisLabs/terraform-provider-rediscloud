---
page_title: "Provider: Redis Enterprise Cloud"
description: |-
   The Redis Enterprise Cloud provider is used to interact with the resources supported by Redis Enterprise Cloud. The provider needs to be configured with the proper credentials before it can be used..
---

# Redis Enterprise Cloud Provider

The Redis Enterprise Cloud provider is used to interact with the resources supported by Redis Enterprise Cloud . The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available provider resources and data sources.

## Configure Redis Enterprise Cloud Programmatic Access

In order to setup authentication with the Redis Enterprise Cloud provider a programmatic API key must be generated for Redis Enterprise Cloud. The [Redis Enterprise Cloud documentation](https://redis.io/docs/latest/integrate/terraform-provider-for-redis-cloud/) contains the most up-to-date instructions for creating and managing your key(s) and IP access.

## Example Usage

```hcl
provider "rediscloud" {
}

# Example resource configuration
resource "rediscloud_subscription" "example" {
  # ...
}
```

Iris-only bearer authentication may omit both Cloud API credentials. A partial
Cloud credential pair is invalid, and Cloud API resources still require both.

## Argument Reference

In addition to [generic `provider` arguments](https://www.terraform.io/docs/configuration/providers.html)
(e.g. `alias` and `version`), the following arguments are supported in the Redis Cloud
`provider` block:

* `api_key` - (Optional) This is the Redis Enterprise Cloud API key. Required with `secret_key` for Cloud API resources; may also be set by the
`REDISCLOUD_ACCESS_KEY` environment variable.

* `secret_key` - (Optional) This is the Redis Enterprise Cloud API secret key. Required with `api_key` for Cloud API resources; may also be set
by the `REDISCLOUD_SECRET_KEY` environment variable.

## Direct Iris services

The provider also supports separate `rediscloud_iris_*` resources for the Agent
Memory and Identity Service control planes. Iris-only configurations use their
service URLs and bearer tokens and do not need Redis Cloud CAPI credentials.
Cloud gateways that support API-key authentication can instead use the
provider's Redis Cloud key and secret for either service.
See the [direct Iris guide](guides/iris.md) for configuration and usage.

* `iris_agent_memory_url` - (Optional) Agent Memory control-plane base URL, or
  `REDISCLOUD_IRIS_AGENT_MEMORY_URL`.
* `iris_agent_memory_token` - (Optional, Sensitive) Agent Memory bearer token, or
  `REDISCLOUD_IRIS_AGENT_MEMORY_TOKEN`.
* `iris_identity_url` - (Optional) Identity Service base URL, or
  `REDISCLOUD_IRIS_IDENTITY_URL`.
* `iris_identity_token` - (Optional, Sensitive) Identity Service bearer token, or
  `REDISCLOUD_IRIS_IDENTITY_TOKEN`.
