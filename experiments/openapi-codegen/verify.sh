#!/usr/bin/env bash
# Measure the raw output, including known generator failures.
set -euo pipefail
cd "$(dirname "$0")/../.."
experiment=experiments/openapi-codegen
shasum -a 256 -c "$experiment/inputs/SHA256SUMS"
for entry in agent-memory:3 identity:2 identity-adapted:1; do
  service=${entry%:*}
  expected=${entry#*:}
  jq -e --argjson expected "$expected" '.resources | length == $expected' \
    "$experiment/results/$service/provider-code-spec.json" > /dev/null
done
if go test ./experiments/openapi-codegen/_generated/agent-memory/... \
    ./experiments/openapi-codegen/_generated/identity/... \
    ./experiments/openapi-codegen/_generated/identity-adapted/... \
    > "$experiment/results/compile.log" 2>&1; then
  echo 'All generated packages compile.'
else
  echo 'Generated package compilation failed; see results/compile.log.'
fi
go run ./experiments/openapi-codegen/_probe | tee "$experiment/results/schema-probe.log"
