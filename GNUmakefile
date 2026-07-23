default: ci

PROVIDER_HOSTNAME=registry.terraform.io
PROVIDER_NAMESPACE=RedisLabs
PROVIDER_TYPE=rediscloud
PROVIDER_TARGET=$(shell go env GOOS)_$(shell go env GOARCH)
PROVIDER_VERSION=99.99.99

PLUGINS_PATH=~/.terraform.d/plugins
PLUGINS_PROVIDER_PATH=$(PROVIDER_HOSTNAME)/$(PROVIDER_NAMESPACE)/$(PROVIDER_TYPE)/$(PROVIDER_VERSION)/$(PROVIDER_TARGET)

BIN=$(CURDIR)/bin
CI_PLUGIN_DIR=$(BIN)/terraform-plugin-dir
CI_SCHEMA_DIR=$(BIN)/providers-schema
RELEASE_NOTES_FILE=$(CURDIR)/release-notes.md

# Use a parallelism of 6 by default for tests, overriding whatever GOMAXPROCS is set to.
TEST_PARALLELISM?=6

.PHONY: default \
        build clean install-local \
        fmt fmt-golangci fmt-terraform \
        lint lint-golangci lint-terraform lint-tfproviderlint lint-docs lint-goreleaser lint-markdown \
        ci govulncheck go-mod-tidy go-build go-build-tests go-unit-test terraform-providers-schema \
        testacc testacc-essentials testacc-check \
        sweep \
        release-notes release

bin:
	mkdir -p $(BIN)

clean:
	@echo "Deleting local provider binary"
	rm -rf $(BIN)

build: bin lint-golangci
	@echo "Building local provider binary"
	go build -o $(BIN)/terraform-provider-rediscloud_v$(PROVIDER_VERSION)
	$(CURDIR)/scripts/generate-dev-overrides.sh

fmt: fmt-golangci fmt-terraform

fmt-golangci:
	@echo "Formatting Go files"
	golangci-lint fmt

fmt-terraform:
	@echo "Formatting Terraform files"
	terraform fmt -recursive

lint: lint-golangci lint-tfproviderlint lint-terraform lint-docs lint-goreleaser lint-markdown

lint-golangci:
	@echo "Running golangci-lint"
	golangci-lint run

lint-tfproviderlint:
	# XS001 — disables "schema should configure Description"
	# XS002 — disables "schema attributes should be in alphabetical order"
	@echo "Running tfproviderlint"
	tfproviderlintx $(TFPROVIDERLINT_ARGS) -XS001=false -XS002=false ./...

lint-terraform:
	@echo "Checking Terraform formatting"
	terraform fmt -check -recursive

lint-docs:
	@echo "Validating documentation"
	tfplugindocs validate

lint-goreleaser:
	@echo "Checking GoReleaser config"
	goreleaser check

lint-markdown:
	@echo "Linting Markdown"
	markdownlint-cli2

ci: govulncheck go-mod-tidy lint go-build go-build-tests go-unit-test terraform-providers-schema
	@echo "All local CI checks passed"

govulncheck:
	@echo "Running govulncheck"
	govulncheck ./...

go-mod-tidy:
	@echo "Checking go.mod/go.sum are tidy"
	go mod tidy -diff

go-build: bin
	@echo "Building provider into local plugin mirror"
	mkdir -p $(CI_PLUGIN_DIR)/$(PLUGINS_PROVIDER_PATH)
	go build -o $(CI_PLUGIN_DIR)/$(PLUGINS_PROVIDER_PATH)/terraform-provider-rediscloud .

go-build-tests:
	@echo "Building test packages"
	go test ./... -run="^$$"

go-unit-test:
	@echo "Running unit tests"
	go test ./... -run="^TestUnit"

terraform-providers-schema: go-build
	@echo "Generating provider schema"
	rm -rf $(CI_SCHEMA_DIR) && mkdir -p $(CI_SCHEMA_DIR)
	printf 'terraform {\n  required_providers {\n    rediscloud = {\n      source  = "$(PROVIDER_NAMESPACE)/$(PROVIDER_TYPE)"\n      version = "$(PROVIDER_VERSION)"\n    }\n  }\n}\n' > $(CI_SCHEMA_DIR)/providers.tf
	echo 'resource "rediscloud_subscription" "example" {}' > $(CI_SCHEMA_DIR)/example.tf
	cd $(CI_SCHEMA_DIR) && terraform init -plugin-dir $(CI_PLUGIN_DIR) && terraform providers schema -json > schema.json
	@echo "Schema written to $(CI_SCHEMA_DIR)/schema.json"

# Fail fast when TEST_PATTERN matches no tests: `go test -run` exits 0 even on no
# match, so a typo'd pattern would silently "pass" without running anything.
# `-list` applies the same regex but only prints matching test names (it runs
# nothing, so no credentials are needed) — CI runs this as its own step before
# provisioning a test account.
testacc-check:
ifndef TEST_PATTERN
	$(error TEST_PATTERN is not set, e.g. make testacc-check TEST_PATTERN='TestAccResourceRedisCloudProSubscription_CRUDI')
endif
	@if [ -z "$$(go test ./... -list='$(TEST_PATTERN)' | grep '^Test')" ]; then \
		echo "Error: no tests match TEST_PATTERN='$(TEST_PATTERN)'"; \
		exit 1; \
	fi

# `-p=1` added to avoid testing packages in parallel which causes `go test` to not stream logs as they are written
testacc: bin
ifndef TEST_PATTERN
	$(error TEST_PATTERN is not set, e.g. make testacc TEST_PATTERN='TestAccResourceRedisCloudProSubscription_CRUDI')
endif
	TF_ACC=1 go test ./... -v -run='$(TEST_PATTERN)' -timeout 360m -p=1 -parallel=$(TEST_PARALLELISM) -coverprofile $(BIN)/coverage.out

# Essentials tests must run serially due to an API limit of one essentials db per
# account, so run them through `testacc` with parallelism pinned to 1.
testacc-essentials:
	$(MAKE) testacc TEST_PARALLELISM=1 TEST_PATTERN='TestAccResourceRedisCloudEssentials|TestAccDataSourceRedisCloudEssentials'

install-local: build
	@echo "Installing local provider binary to plugins mirror path $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)"
	mkdir -p $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)
	cp $(BIN)/terraform-provider-rediscloud_v$(PROVIDER_VERSION) $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test ./provider -v -sweep=ALL $(SWEEPARGS) -timeout 30m

release-notes:
	@echo "Generating release notes"
	./scripts/release-notes.sh > $(RELEASE_NOTES_FILE)
	@echo "===== release notes ====="
	@cat $(RELEASE_NOTES_FILE)
	@echo "================================="

release: release-notes
	goreleaser release --clean --release-notes=$(RELEASE_NOTES_FILE)
