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

.PHONY: build clean testacc install_local website website-test tfproviderlint

build:
	@echo "Building local provider binary"
	@mkdir -p ./bin
	go build -o bin/terraform-provider-rediscloud_v$(PROVIDER_VERSION)

clean:
	@echo "Deleting local provider binary"
	rm -rf ./bin

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m -parallel=$(TEST_PARALLELISM)

install_local: build
	@echo "Installing local provider binary to plugins mirror path $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)"
	@mkdir -p $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)
	@cp ./bin/terraform-provider-rediscloud_v$(PROVIDER_VERSION) $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test ./internal/provider -v -sweep=ALL $(SWEEPARGS) -timeout 30m

tfproviderlint:
	@go run github.com/bflad/tfproviderlint/cmd/tfproviderlint ./...
