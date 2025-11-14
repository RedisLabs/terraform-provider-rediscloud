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
- Update the CHANGELOG if needed (follow existing format)
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

# Create an annotated tag with the version number
# Use semantic versioning: v<major>.<minor>.<patch>
git tag v1.2.3

# Push the tag to origin
git push origin v1.2.3
```

**Important:** The tag MUST follow the pattern `v*` (e.g., `v1.2.3`, `v0.5.0`) for the release automation to trigger.

### 5. Automation Takes Over

When you push the tag, the `release` workflow (`.github/workflows/release.yml`) automatically:

1. Checks out the tagged commit
2. Sets up Go using the version from `go.mod`
3. Imports the GPG signing key from GitHub secrets
4. Runs [GoReleaser](https://goreleaser.com/) to:
   - Build binaries for all supported platforms
   - Sign the binaries with GPG
   - Create a GitHub release
   - Publish to the Terraform Registry

**No manual intervention required** - just wait for the workflow to complete (usually 5-10 minutes).

### 6. Create a GitHub Release

To do this:

2. Go to https://github.com/RedisLabs/terraform-provider-rediscloud/releases
2. Add a new draft release
3. Update the description with your entries in the `CHANGELOG.md`.

**This step is not required for the release to work** - it's just nice to have for users.

## Additional Resources

- [Terraform Registry Provider Publishing](https://www.terraform.io/docs/registry/providers/publishing.html)
- [GoReleaser Documentation](https://goreleaser.com/intro/)
- [Semantic Versioning](https://semver.org/)
