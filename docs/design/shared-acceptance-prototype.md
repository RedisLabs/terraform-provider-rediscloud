# Shared acceptance prototype: Terraform

Date: 2026-11-25
Status: Experimental, credential-free only

## TLDR

A simulated adapter consumes the provisional shared scenario contract from one pinned Crossplane checkout and runs the actual Terraform CLI and source-built provider against a loopback fake. It exercises the complete ordered Essentials lifecycle and optional eviction update in an isolated in-memory account. Results are simulated claims requiring verification; they do not establish live Cloud coverage.

## Problem and goal

The native `TestAccResourceRedisCloudEssentialsDatabase_CRUDI` exercises create, state reads, name update and import; it does not assert the shared contract's exact database 404 while the subscription remains present, eviction update with stable identity, account-wide admission, and complete final inventory. A successful native acceptance test must not be relabeled as a complete shared lifecycle without those observations and independent evidence.

The goal is to consume the same scenario document without copying scenario definitions. The partial import/update fixture remains available; a separate local fake supplies the full create/delete lifecycle. This is not a live adapter and has no capability to create or delete Cloud resources. Crossplane's repository is a provisional host; ownership and neutral distribution are undecided.

## Behavior and safety

The runner supports linux/amd64 only. It requires a clean Crossplane checkout at the pinned revision and verifies the contract and scenario hashes. It builds an archived exact Terraform source commit into a temporary local Terraform plugin directory, invokes the local loopback fixture with dummy credentials, and reports the binary digest. The plugin-directory version label is required by the fixture and is not a release claim. Only a small environment allowlist is passed; Go module network access is disabled. No real credentials, account, lock or recovery state is used.

The full fake fixture executes real Terraform apply operations in the contract's order, with fake credentials and an HTTP server bound to loopback only. It checks exact fake database absence while the fake parent remains present, then exact fake parent absence and an empty fake inventory. It logs ordered fake observations and fake write paths. In GitHub Actions, receipts reference the Actions console and classify only as `simulated_claim_requires_verification`; a local invocation without a hosted evidence URL reports `unsupported`. The fake server is isolated in memory and supplies neither a real Cloud account nor an account-wide lock.

## Validation and open work

Run from the Terraform repository on linux/amd64, with dependencies for the pinned source and fixture already cached:

```
python3 scripts/shared_acceptance/terraform_lifecycle_adapter.py --source-repo <terraform-repo> --fixture-repo <crossplane-pinned-checkout> --go <go-binary> --terraform <terraform-binary> --scenario essentials-database-eviction-update
```

The runner reports pinned source/fixture revisions and binary digests; a failing build, test, pin or hash returns an error, not a pass. Live receipt generation, isolated account admission, trustworthy live read/cleanup evidence, source distribution and a live writer adapter remain unimplemented. No production rollout or rollback applies to this experimental artifact; delete the prototype if its provisional contract is superseded.

ADR: no durable shared-hosting or architecture decision is made here. BDR: no business or support commitment changes.
