# Pro database resource on the plugin framework

Ticket: [RED-202074](https://redislabs.atlassian.net/browse/RED-202074)
Epic: RED-202040
Date: 2026-11-06
Status: proposed with the pull request
Owner: Teodor Stoykov (ticket assignee)

## TLDR

- `rediscloud_subscription_database` moves from SDKv2 to terraform-plugin-framework and stays on the protocol-v5 mux.
- The public name, attribute names, defaults, and state values stay the same. `subscription_id`, `protocol`, and `port` stay the only schema-level replacements.
- `modules` stays a set attribute. SDKv2 used `ConfigModeAttr`, so a nested block would change configuration syntax and the state type.

## Architecture

The mux keeps serving protocol v5. The framework provider owns this resource. The SDKv2 provider no longer registers it.

```text
terraform (protocol v5)
        |
        +-- framework provider
        |     rediscloud_subscription_database
        |         |
        |         +-- ModifyPlan: version, modules, backup, passwordless
        |         +-- Create -> Update -> Read
        |         +-- proDatabaseAPI
        |               |
        |               +-- live adapter -> rediscloud-go-api
        |
        +-- SDKv2 provider
              other resources, including essentials database
```

Create still cannot send every field, so create is followed by update and then read. The Cloud API client is behind `proDatabaseAPI`. Unit tests substitute a fake and do not call Redis Cloud.

`alert` and `remote_backup` stay nested blocks with custom collection types. Primitive lists, sets, and maps stay attributes. `modules` is a set of objects with the same custom set type, attached as an attribute rather than a block.

## Decisions

- Nested blocks are used for `alert` and `remote_backup` because those were SDKv2 blocks. Framework nested attributes need protocol v6 and are not used.
- `modules` stays `modules = [{ name = "..." }]`. A nested block would not match the SDKv2 `ConfigModeAttr` schema. Replacement for a module change is decided in `ModifyPlan`, and only when Redis is below 8. The attribute itself is not `RequiresReplace`, so the Redis 8 suppress can still apply.
- `redis_version` stays optional and computed. Read writes the API version into it only when state has no value. A planned version that the running version already satisfies does not change state. Upgrade runs only when the prior and planned values are both set, differ, and the actual version is not already the planned one. The `TODO(TF3.0)` comment is unchanged.
- Schema `RequiresReplace` is only `subscription_id`, `protocol`, and `port`.
- SDK defaults are optional, computed, and defaulted. The framework requires computed when a default is set. The default values are unchanged, including empty `client_ssl_certificate` and `periodic_backup_path`.
- `memory_limit_in_gb` and `dataset_size_in_gb` stay mutually exclusive. `average_item_size_in_bytes` still conflicts with `ram_percentage`, and `client_ssl_certificate` still conflicts with `client_tls_certificates`. Those checks use configuration, so an omitted default does not conflict.
- `periodic_backup_path` against `remote_backup` is checked in `ModifyPlan` because one is an attribute and one is a block. The `time_utc` error text is unchanged.
- A 12-hour backup whose API time is 12 hours from the configured time keeps the configured time.
- Source IP defaults stay: empty, `0.0.0.0/0`, or the private ranges. Read keeps a custom list when the API still reports a default. Update sends the subscription default when the configured list is a default.
- `password` stays sensitive, optional, and computed, without `UseStateForUnknown`. Turning passwordless off without a password still returns `when disabling passwordless mode, you must provide a 'password'`.
- Read removes the resource when the database is missing. Delete reports that error instead of treating it as success.
- Create and update write state once the id is known, including when a later wait fails, so Terraform keeps the remote database.
- An omitted TLS certificate list stays null. Redis 8 read leaves `modules` unset when the prior set was empty, and refreshes it from the API when the user configured modules.

`ToDatabaseId`, the flatten helpers, and the tag helpers stay exported. Essentials and the pro data source still call them.

## Implementation

- `provider/pro/resource_pro_database.go` and the model, type, behavior, mapping, API, and CRUD files replace `resource_rediscloud_pro_database.go`.
- `provider/framework_provider.go` registers `pro.NewProDatabaseResource`.
- `provider/sdk_provider.go` drops the SDKv2 registration.
- `provider/utils/redis_version.go` exposes `RedisVersionRequestSatisfied` so the SDKv2 diff suppress and the framework plan use one comparison.

## Testing

- Unit tests cover the schema, custom types, plan rules, read mapping, validators, and create/read/update/delete against a fake API.
- New pro database statements are 82.2% covered (801/975). `RedisVersionRequestSatisfied` is covered by the existing utils test.
- `TestProvider`, `TestFrameworkProvider`, and `TestMuxServer` load the mux schema, including this resource.
- `gofmt` and `golangci-lint fmt --diff` report no diff. `go build -buildvcs=false ./...` and `go vet` on the changed packages pass.
- Acceptance tests were not run. They call the live Redis Cloud API.

## Operations / follow-ups

- Run the existing pro database acceptance suite before release. It checks `modules = [{ name = "..." }]`, `redis_version`, empty certificate and backup strings, tags, and the `time_utc` error.
- The `TODO(TF3.0)` notes stay: stop writing `redis_version` from read, and make the empty-string certificate, backup, and endpoint fields nullable.
- The live API adapter is a thin wrapper and is not covered by the unit tests.
