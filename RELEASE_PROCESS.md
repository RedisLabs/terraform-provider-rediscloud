# Release Process

This document describes how to release a new version of `terraform-provider-rediscloud`.

## Prerequisites

You need:
- **Push access to GitHub** - ability to push tags to the repository
- **PR permissions** - ability to submit and merge pull requests

That's it! The release automation handles everything else (building binaries, GPG signing, publishing to the Terraform Registry).

## Release Steps

### 1. Open a Pull Request to `main`

Create a PR with your changes targeting the `main` branch. The PR should:
- Include all the changes you want to release
- Add a new entry to [`CHANGELOG.md`](./CHANGELOG.md) for this version, following the existing `## MAJOR.MINOR.PATCH (Month Day Year)` format. **This is required and must be merged before you tag** — the release automation generates the GitHub release notes from the latest CHANGELOG entry in the tagged commit.
- Pass all required checks before merging

You should get someone to test and review your changes manually before releasing.

### 2. Wait for Smoke Tests to Pass

The PR will automatically run smoke tests via the `Terraform Provider Checks - PR workflow` workflow. 

The smoke tests consist of acceptance tests that will check the major resources to ensure that your changes did not cause any major regressions. They will typically take between 30 and 60 minutes.

Additionally, these checks must pass:
- `go build` - ensures the provider compiles
- `tfproviderlint` - Terraform provider linting
- `terraform providers schema` - validates schema generation
- `go unit test` - runs unit tests

**Do not merge until all smoke tests pass.**

### 3. Merge the Pull Request

Once the smoke tests pass, merge the PR to `main`.

### 4. Tag and Push

After merging to `main`:

```bash
# Switch to main and pull the merged changes
git checkout main
git pull origin main

# Create a tag with the version number
# Use semantic versioning: v<major>.<minor>.<patch>
git tag v1.2.3

# Push the tag to origin
git push origin v1.2.3
```

**Important:** The tag MUST follow the pattern `v*` (e.g., `v1.2.3`, `v0.5.0`) for the release automation to trigger.

### 5. Automation Takes Over

When you push the tag, the `release` workflow (`.github/workflows/release.yml`) automatically:

1. Checks out the tagged commit
2. Sets up the pinned Nix toolchain (via the `./.github/actions/setup-nix` action)
3. Imports the GPG signing key from GitHub secrets
4. Runs [GoReleaser](https://goreleaser.com/) (`make release`) to:
   - Generate the release notes from the latest `CHANGELOG.md` entry
   - Build binaries for all supported platforms
   - Sign the binaries with GPG
   - Create the GitHub release (published, not a draft)
   - Publish to the Terraform Registry

**No manual intervention required** - just wait for the workflow to complete (usually 5-10 minutes).

### 6. Verify the Release

GoReleaser creates the GitHub release automatically, with its notes taken from the latest
`CHANGELOG.md` entry — there is no manual release-creation step. Once the workflow finishes:

1. Check the [releases page](https://github.com/RedisLabs/terraform-provider-rediscloud/releases) to confirm the release and its notes look correct.
2. Wait for the [Terraform Registry](https://registry.terraform.io/providers/RedisLabs/rediscloud/latest) to pick up the new version (usually a few minutes).

## Additional Resources

- [Terraform Registry Provider Publishing](https://www.terraform.io/docs/registry/providers/publishing.html)
- [GoReleaser Documentation](https://goreleaser.com/intro/)
- [Semantic Versioning](https://semver.org/)
