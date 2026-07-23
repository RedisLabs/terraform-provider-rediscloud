default: testacc

PROVIDER_HOSTNAME=registry.terraform.io
PROVIDER_NAMESPACE=RedisLabs
PROVIDER_TYPE=rediscloud
PROVIDER_TARGET=$(shell go env GOOS)_$(shell go env GOARCH)
PROVIDER_VERSION = 99.99.99

PLUGINS_PATH = ~/.terraform.d/plugins
PLUGINS_PROVIDER_PATH=$(PROVIDER_HOSTNAME)/$(PROVIDER_NAMESPACE)/$(PROVIDER_TYPE)/$(PROVIDER_VERSION)/$(PROVIDER_TARGET)

BIN=$(CURDIR)/bin
CI_PLUGIN_DIR = $(BIN)/terraform-plugin-dir
CI_SCHEMA_DIR = $(BIN)/providers-schema
RELEASE_NOTES_FILE = release-notes.md

# Use a parallelism of 6 by default for tests, overriding whatever GOMAXPROCS is set to.
TEST_PARALLELISM?=6
TESTARGS?=-short

.PHONY: build clean fmt fmt-golangci fmt-terraform lint lint-golangci lint-terraform lint-tfproviderlint tfproviderlint \
        testacc testacc-essentials install-local sweep sweep-prefix \
        lint-docs lint-goreleaser ci go-mod-tidy govulncheck go-unit-test go-build go-build-tests \
        terraform-providers-schema lint-markdown release-notes release

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

# `-p=1` added to avoid testing packages in parallel which causes `go test` to not stream logs as they are written
testacc: bin
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 360m -p=1 -parallel=$(TEST_PARALLELISM) -coverprofile $(BIN)/coverage.out

# Essentials tests must run serially due to API limitation of one essentials db per account
testacc-essentials: bin
	TF_ACC=1 go test ./provider -v -run="TestAccResourceRedisCloudEssentials|TestAccDataSourceRedisCloudEssentials" -timeout 360m -p=1 -parallel=1 -coverprofile $(BIN)/coverage.out

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
