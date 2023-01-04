default: testacc

PROVIDER_HOSTNAME=registry.terraform.io
PROVIDER_NAMESPACE=RedisLabs
PROVIDER_TYPE=rediscloud
PROVIDER_TARGET=$(shell go env GOOS)_$(shell go env GOARCH)
PROVIDER_VERSION = 99.99.99

PLUGINS_PATH = ~/.terraform.d/plugins
PLUGINS_PROVIDER_PATH=$(PROVIDER_HOSTNAME)/$(PROVIDER_NAMESPACE)/$(PROVIDER_TYPE)/$(PROVIDER_VERSION)/$(PROVIDER_TARGET)

# Use a parallelism of 3 by default for tests, overriding whatever GOMAXPROCS is set to.
TEST_PARALLELISM?=3
TESTARGS?=-short

BIN=$(CURDIR)/bin
$(BIN)/%:
	@echo "Installing tools from tools/tools.go"
	@cat tools/tools.go | grep _ | awk -F '"' '{print $$2}' | GOBIN=$(BIN) xargs -tI {} go install {}

.PHONY: build clean testacc install_local website website-test tfproviderlint

build:
	@echo "Building local provider binary"
	@mkdir -p ./$(BIN)
	go build -o ./$(BIN)/terraform-provider-rediscloud_v$(PROVIDER_VERSION)
	@sh -c "'$(CURDIR)/scripts/generate-dev-overrides.sh'"

clean:
	@echo "Deleting local provider binary"
	rm -rf $(BIN)

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 360m -parallel=$(TEST_PARALLELISM)

install_local: build
	@echo "Installing local provider binary to plugins mirror path $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)"
	@mkdir -p $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)
	@cp ./$(BIN)/terraform-provider-rediscloud_v$(PROVIDER_VERSION) $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test ./internal/provider -v -sweep=ALL $(SWEEPARGS) -timeout 30m

tfproviderlintx: $(BIN)/tfproviderlint
	$(BIN)/tfproviderlintx $(TFPROVIDERLINT_ARGS) ./...

tfproviderlint: $(BIN)/tfproviderlintx
	$(BIN)/tfproviderlint $(TFPROVIDERLINT_ARGS) ./...
