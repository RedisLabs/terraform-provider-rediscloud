# Repository guide

Use this file for conventions that apply across the repository. Put procedures
for a particular kind of work in a skill under `.agents/skills/` so they are
loaded only when they are relevant.

## What this repository contains

`terraform-provider-rediscloud` is a Go Terraform provider. The provider is
being moved incrementally from `terraform-plugin-sdk/v2` to
`terraform-plugin-framework`. Until that migration finishes, the SDK and
Framework providers run together behind a protocol-v5 mux.

The main directories are:

- `provider/` contains provider code and tests. Framework implementations are
  normally grouped by resource family.
- `docs/` contains Terraform Registry documentation. Do not put internal
  engineering notes there.
- `playground/` is the gitignored location for manual Terraform configurations.
- `scripts/` contains repository automation.
- `bin/` receives local build output and the Terraform development override.

## Build and test commands

- Run `make build` to compile the provider and create
  `bin/developer_overrides.tfrc`.
- Run `make fmtcheck` to check formatting with the repository's goimports
  settings.
- Run `go build ./...` and `go vet ./...` for repository-wide compilation and
  static checks.
- Run `go test ./...` for unit tests.
- Run `make testacc` only with a configured Redis Cloud test account. Acceptance
  tests set `TF_ACC=1` and make real API calls.

If the default Go build cache is not writable, point `GOCACHE` at a writable
directory under `/tmp`. Terraform-based tests may also need to run outside a
restricted command sandbox because they start a provider process and communicate
with it over a local socket.

For manual testing, build the provider and then export:

```sh
export TF_CLI_CONFIG_FILE="$PWD/bin/developer_overrides.tfrc"
```

Terraform will then use the local provider binary for configurations under
`playground/`.

## Branches and pull requests

Base ordinary work on the newest active `release/<x.y.z>` branch and target the
same branch with the pull request. Do not assume that a release branch remembered
from an earlier task is still current. Fetch first, list `origin/release/*`, and
compare their semantic versions. Ask when multiple branches appear active or the
newest branch already looks released.

Work may be parked on `main` before its release branch exists. Move that work to
the appropriate release branch once the branch is created. Feature branches do
not normally target `main`; the release branch itself is what later merges into
`main`.

For stacked work, base each child branch and pull request on its immediate parent
branch. Verify that helpers or packages expected from the parent are actually
present before starting the child migration.

`.github/workflows/terraform_provider_pr.yml` defines the branches for which pull
request checks run. `RELEASE_PROCESS.md` describes the separate process of
merging a completed release branch into `main`.

## System ownership

The provider calls the public Redis Cloud API through
`github.com/RedisLabs/rediscloud-go-api`. Several services participate behind
that API, and their ownership matters when behaviour differs between the public
API and the console:

- Subscription Manager, also called SM or Garantia, owns the primary data and
  the core account, subscription, database, and cluster behaviour.
- `sm-cloud-api`, commonly called CAPI, is the public API gateway. It reads from
  an SM database replica and sends asynchronous writes to SM.
- `sm-das` provides a separate GraphQL read interface over SM-owned data for
  internal tooling.
- RCP provisions dedicated clusters in customer cloud accounts and synchronises
  their state with SM.
- `cloud-ui` is the web console and calls SM's session-authenticated API rather
  than CAPI.
- `rediscloud-go-api` is the Go client used by this provider to reach CAPI.

The console and this provider therefore exercise different read paths. Do not
infer public API behaviour solely from the console. In particular, API list
responses can arrive in an unspecified order even when the console's backend
sorts the same data. Terraform code that stores an ordered collection must impose
a stable order itself.

When investigating backend behaviour, first look for an existing local checkout
of the owning service and verify claims against its code. If a required checkout
cannot be found, ask the user for its location or for the missing repository to
be made available. Do not clone a service repository solely for the current work
session. Architecture prose can lag behind deployed behaviour.

## Task skills

Recurring workflows live as Agent Skills:

| Skill | Use it for |
| --- | --- |
| `migrate-to-plugin-framework` | Migrating an SDKv2 resource or data source, or adding a new Framework implementation. |

Skills are stored at `.agents/skills/<skill-name>/SKILL.md`. When adding or
changing a skill, validate it with the available Agent Skills validator.

## Maintaining repository guidance

- Keep repository-wide facts here and task-specific procedures in their skill.
- Record durable decisions and non-obvious failure modes, not a diary of one
  implementation.
- Replace an outdated rule where it lives instead of adding a contradictory
  exception later in the file.
- Prefer guidance that explains non-obvious reasoning and helps future
  contributors choose the right approach. Do not repeat implementation details
  that are clear from the relevant code.
- Recheck commands, paths, and precedents whenever you rely on them. Documentation
  should be corrected in the same change that proves it stale.
