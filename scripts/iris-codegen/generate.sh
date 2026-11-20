#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
shasum -a 256 -c scripts/iris-codegen/inputs/SHA256SUMS
go run ./internal/irisgen
for service in agent-memory identity; do
  go run github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@v0.3.0 generate \
    --config "scripts/iris-codegen/prepared/$service.config.json" \
    --output "scripts/iris-codegen/prepared/$service.spec.json" \
    "scripts/iris-codegen/prepared/$service.openapi.json"
done
go run ./internal/irisgen adapt
for service in agent-memory identity; do
  go run github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@v0.4.1 generate resources \
    --input "scripts/iris-codegen/prepared/$service.adapted.json" \
    --output provider/iris/generated
done
goimports -local github.com/RedisLabs/terraform-provider-rediscloud -w provider/iris/generated provider/iris/schemas_gen.go
