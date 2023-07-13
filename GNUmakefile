default: testacc

PROVIDER_HOSTNAME=registry.terraform.io
PROVIDER_NAMESPACE=RedisLabs
PROVIDER_TYPE=rediscloud
PROVIDER_TARGET=$(shell go env GOOS)_$(shell go env GOARCH)
PROVIDER_VERSION = 99.99.99

PLUGINS_PATH = ~/.terraform.d/plugins
PLUGINS_PROVIDER_PATH=$(PROVIDER_HOSTNAME)/$(PROVIDER_NAMESPACE)/$(PROVIDER_TYPE)/$(PROVIDER_VERSION)/$(PROVIDER_TARGET)

# Use a parallelism of 4 by default for tests, overriding whatever GOMAXPROCS is set to.
TEST_PARALLELISM?=4
TESTARGS?=-short

bin:
	@mkdir -p bin/

BIN=$(CURDIR)/bin
$(BIN)/%:
	@echo "Installing tools from tools/tools.go"
	@cat tools/tools.go | grep _ | awk -F '"' '{print $$2}' | GOBIN=$(BIN) xargs -tI {} go install {}

.PHONY: build clean testacc generate_coverage install_local website website-test tfproviderlint

build: bin
	@echo "Building local provider binary"
	go build -o $(BIN)/terraform-provider-rediscloud_v$(PROVIDER_VERSION)
	@sh -c "'$(CURDIR)/scripts/generate-dev-overrides.sh'"

clean:
	@echo "Deleting local provider binary"
	rm -rf $(BIN)

# `-p=1` added to avoid testing packages in parallel which causes `go test` to not stream logs as they are written
testacc: bin
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 360m -p=1 -parallel=$(TEST_PARALLELISM) -coverprofile bin/coverage.out

generate_coverage:
	go tool cover -html=bin/coverage.out -o bin/coverage.html

install_local: build
	@echo "Installing local provider binary to plugins mirror path $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)"
	@mkdir -p $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)
	@cp $(BIN)/terraform-provider-rediscloud_v$(PROVIDER_VERSION) $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test ./provider -v -sweep=ALL $(SWEEPARGS) -timeout 30m

tfproviderlintx: $(BIN)/tfproviderlintx
	$(BIN)/tfproviderlintx $(TFPROVIDERLINT_ARGS) ./...

tfproviderlint: $(BIN)/tfproviderlint
	$(BIN)/tfproviderlint $(TFPROVIDERLINT_ARGS) ./...
