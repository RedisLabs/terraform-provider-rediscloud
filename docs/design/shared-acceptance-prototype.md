# Shared acceptance prototype: Terraform

Date: 2026-11-25
Status: Experimental, credential-free only

## TLDR

A partial adapter consumes the provisional shared scenario contract from one pinned Crossplane checkout and runs its existing loopback fixture using a Terraform provider built from a pinned source revision. It always reports the shared scenario as unsupported because the fixture starts with an imported fake database and does not perform the complete lifecycle.

## Problem and goal

The native `TestAccResourceRedisCloudEssentialsDatabase_CRUDI` exercises create, state reads, name update and import; it does not assert the shared contract's exact database 404 while the subscription remains present, eviction update with stable identity, account-wide admission, and complete final inventory. A successful native acceptance test must not be relabeled as a complete shared lifecycle without those observations and independent evidence.

The goal is to reuse the real existing fake-service test and scenario document without copying either test or scenario definitions. It is not a live adapter and has no capability to create or delete Cloud resources. Crossplane's repository is a provisional host; ownership and neutral distribution are undecided.

## Behavior and safety

The runner supports linux/amd64 only. It requires a clean Crossplane checkout at the pinned revision and verifies the contract and scenario hashes. It builds an archived exact Terraform source commit into a temporary local Terraform plugin directory, invokes the existing loopback fixture with dummy credentials, and reports the binary digest. The plugin-directory version label is required by the fixture and is not a release claim. Only a small environment allowlist is passed; Go module network access is disabled. No credentials, account, lock or recovery state is used.

The fake fixture exercises a real Terraform CLI, provider and SDK: repeated unchanged plans, a failed then successful update of an imported database's eviction policy, and stable fake identity. It does not create a subscription or database, prove exact deletion or authenticate a complete inventory. Its `unsupported` classification intentionally cannot produce a full scenario pass, even when the fake test passes. The fake server is owned by the existing fixture, not by a shared Cloud account.

## Validation and open work

Run from the Terraform repository on linux/amd64, with dependencies for the pinned source and fixture already cached:

```
python3 scripts/shared_acceptance/terraform_adapter.py --source-repo <terraform-repo> --fixture-repo <crossplane-pinned-checkout> --go <go-binary> --terraform <terraform-binary>
```

The runner reports pinned source/fixture revisions and binary digests; a failing build, test, pin or hash returns an error, not a pass. Actual receipt generation, isolated account admission, trustworthy read/cleanup evidence, source distribution and a live writer adapter remain unimplemented. No production rollout or rollback applies to this experimental read-only artifact; delete the prototype if its provisional contract is superseded.

ADR: no durable shared-hosting or architecture decision is made here. BDR: no business or support commitment changes.
