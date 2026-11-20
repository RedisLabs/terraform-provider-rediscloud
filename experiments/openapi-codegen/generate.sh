#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
experiment=experiments/openapi-codegen
for service in agent-memory identity identity-adapted; do
  input_service=${service%-adapted}
  mkdir -p "$experiment/results/$service"
  go run github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@v0.3.0 generate \
    --config "$experiment/config/$service.yml" \
    --output "$experiment/results/$service/provider-code-spec.json" \
    "$experiment/inputs/$input_service.openapi.json" \
    2>&1 | tee "$experiment/results/$service/openapi.log"
  go run github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@v0.4.1 generate resources \
    --input "$experiment/results/$service/provider-code-spec.json" \
    --output "$experiment/_generated/$service" \
    2>&1 | tee "$experiment/results/$service/framework.log"
done
