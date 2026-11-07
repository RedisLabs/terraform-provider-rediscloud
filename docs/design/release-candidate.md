# Release candidate before the official provider publish

Date: 2026-11-06
Status: Proposed
Ticket: none
Design: this file
ADR: none. The publish path stays the existing GoReleaser tag workflow. A candidate is a second entry to that path, marked as a GitHub pre-release.
BDR: none.

## TLDR

Actions → Release-candidate builds the selected commit and publishes `vX.Y.Z-rcN` as a GitHub pre-release. The Terraform Registry does not publish it. Test Automation installs the zip from that pre-release and runs the private-connectivity and ACL specs before the official `vX.Y.Z` tag.

## Problem

Ofer's Test Automation specs install `RedisLabs/rediscloud` from the Terraform Registry. Until the official tag is pushed, that version does not exist, so the specs cannot run against the build that is about to be published.

## Goals

- Publish a candidate with the official version plus `-rc` and a number, for example `2.21.0-rc1`.
- Keep that candidate off the Terraform Registry and off the latest-release pin that Crossplane and Pulumi follow.
- Give Test Automation a signed zip they can install before the official tag.

## Non-goals

- Replacing the official release. The stable tag `vX.Y.Z` is still the registry publish.
- Making the candidate private. The provider repository is public, so the pre-release assets are readable. What stays off the public install path is the registry version.
- Changing the Test Automation specs in `cloud-automation` in this change. They still need a filesystem mirror and an exact version pin.
- Opening a Crossplane or Pulumi pull request.

## Behavior

`.github/workflows/release-candidate.yml` is manual. The version input must match `X.Y.Z-rcN`. The workflow checks out the chosen ref, signs with the same GPG key as the official release, and runs `make release-candidate`.

`make release-candidate` tags `vX.Y.Z-rcN` on that commit and runs GoReleaser with `.goreleaser-rc.yml` merged over `.goreleaser.yml`. The overlay sets `prerelease: true` and `make_latest: false`.

`.github/workflows/release.yml` still triggers on `v*`, and its job skips any tag whose name contains a hyphen. A candidate tag therefore does not run the registry publish.

Test Automation downloads `terraform-provider-rediscloud_<version>_linux_amd64.zip` and installs the binary under `registry.terraform.io/RedisLabs/rediscloud/<version>/linux_amd64/`. Terraform must use that filesystem mirror and `version = "<version>"`.

## Validation

The version check rejects a stable version and a `v` prefix. A candidate run is proven when the GitHub release is marked pre-release, the registry page for that version does not exist, and `/releases/latest` is still the previous stable tag.

## Rollout

These files are local and uncommitted. They take effect only after they are on the default branch, because Actions → Run workflow reads `workflow_dispatch` from that branch. The skip in `release.yml` has to ship in the same change. Without it, creating the candidate tag would also start the official publish.

## Open questions

- Env0 still has to be told to use the filesystem mirror. This workflow only produces the zip.
- The first candidate number for a given version is chosen by the person who runs the workflow.
