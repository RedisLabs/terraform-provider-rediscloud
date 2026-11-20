# Input provenance

Copied without modification from the local `iris/main` checkout at commit
`23ff21223ab3b7a06c37e5fc1c4e1a3564f6d442`. Both source files were clean in Git.

- `agent-memory.openapi.json`:
  `generated/smithy/public/cp-memory/MemoryControlPlaneService.openapi.json`
- `identity.openapi.json`:
  `generated/smithy/source/ids-control/IdentityControlService.openapi.json`

The memory input is the public direct-service control-plane specification.
Identity uses the source control-plane specification; this checkout has no
corresponding identity file under `generated/smithy/public/`.
Neither input is the Redis Cloud CAPI specification.

`SHA256SUMS` records the input content; verify it from the provider checkout root.
