---
page_title: "Direct Iris control planes"
description: |-
  Configure and manage Agent Memory and Identity Service through their direct APIs.
---

# Direct Iris control planes

The `rediscloud_iris_*` resources call the Agent Memory and Identity Service
control planes directly. Configure each service with its own base URL and bearer
token. The base URL can include a reverse-proxy path prefix; do not append an
operation path such as `/v1/api-keys`.

```hcl
provider "rediscloud" {
  iris_agent_memory_url = "https://memory-control.example.com"
  iris_identity_url     = "https://identity.example.com"
}
```

Supply tokens using `REDISCLOUD_IRIS_AGENT_MEMORY_TOKEN` and
`REDISCLOUD_IRIS_IDENTITY_TOKEN`, or the sensitive provider attributes
`iris_agent_memory_token` and `iris_identity_token`. URLs can also be supplied
using `REDISCLOUD_IRIS_AGENT_MEMORY_URL` and `REDISCLOUD_IRIS_IDENTITY_URL`.
Only configure the services you use. A configured service URL requires a token.

Iris-only configurations do not require Redis Cloud `api_key` and `secret_key`.
Supply those credentials as usual when also managing Redis Cloud resources.
The existing `rediscloud_agent_memory` and `rediscloud_agent_memory_api_key`
resources continue to use Redis Cloud CAPI and its credentials.

## Resources

- `rediscloud_iris_agent_memory` manages a memory store backed by an existing database.
- `rediscloud_iris_agent_memory_type` manages one custom type within a store.
- `rediscloud_iris_agent_memory_extractor` manages one extractor within a store.
- `rediscloud_iris_identity_api_key` manages a generic Identity Service API key and its grants.
- `rediscloud_iris_identity_provider` manages a tenant's external identity provider.
- `rediscloud_iris_identity_subject_grant_set` manages the subject identity container.
  Its `products_json` is read-only; this resource does not manage individual product grants.

Use the matching resource documentation for configuration examples and imports.
Avoid managing the same remote object through both an Iris and a CAPI resource.

## JSON configuration

Complex settings use `*_json` attributes. Use `jsonencode(...)` with the API's
camelCase property names inside the JSON; top-level Terraform attributes use
snake_case. JSON is validated against the corresponding request shape, and
arbitrary grant `constraints` are preserved.

```hcl
resource "rediscloud_iris_agent_memory" "example" {
  name = "application-memory"
  database_json = jsonencode({
    databaseId = "your-existing-database-id"
  })
  long_term_memory_json = jsonencode({
    ttlSeconds = 86400
  })
}
```

Omitted optional settings use the service's defaults. Omitted properties in
mutable JSON settings leave existing server values unchanged during a PATCH.
The provider retains the configured JSON shape during refresh, excluding extra
response-only defaults and preserving credentials the service cannot return.
Sensitive model JSON is stored in Terraform state. Protect state access as you
would other Terraform-managed credentials.

Store database changes require replacement. Changes to an embedding model or
a model provider, or switching a store from platform models to its own LLM or
embedding model, also require replacement. Credential rotation and supported
same-provider LLM model changes use an update. Identity subjects, parent IDs,
and other create-only fields require replacement.

Memory stores wait for `READY` after creation and update. A failed readiness
check retains the identifier and latest observed status in state. Operations
have a 30-minute deadline; ordinary refresh has a five-minute deadline.
Extractor and memory-type writes include the revision last read from the
service. A conflicting write produces an error for the next refresh and plan
to resolve, rather than retrying with a newer revision automatically.

The Identity API key `token` is sensitive and returned only at creation. It is
retained through refresh and update. Import cannot recover that token or model
credentials, so imported secrets remain absent until separately configured.

## Store deletion

`flush_on_delete` defaults to `false`, retaining short-term sessions when the
store is removed. Set it to `true` to request deletion of those sessions.
Long-term memories are retained by the service in either case. A resource
already missing from the service is removed from Terraform state.
