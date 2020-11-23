default: testacc

PROVIDER_HOSTNAME=registry.terraform.io
PROVIDER_NAMESPACE=RedisLabs
PROVIDER_TYPE=rediscloud
PROVIDER_TARGET=$(shell go env GOOS)_$(shell go env GOARCH)
PROVIDER_VERSION = 99.99.99

PLUGINS_PATH = ~/.terraform.d/plugins
PLUGINS_PROVIDER_PATH=$(PROVIDER_HOSTNAME)/$(PROVIDER_NAMESPACE)/$(PROVIDER_TYPE)/$(PROVIDER_VERSION)/$(PROVIDER_TARGET)

WEBSITE_REPO=github.com/hashicorp/terraform-website
PROVIDER_TYPE=rediscloud

.PHONY: build clean testacc install_local website website-test

build:
	@echo "Building local provider binary"
	@mkdir -p ./bin
	go build -o bin/terraform-provider-rediscloud_v$(PROVIDER_VERSION)

clean:
	@echo "Deleting local provider binary"
	rm -rf ./bin

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

install_local: build
	@echo "Installing local provider binary to plugins mirror path $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)"
	@mkdir -p $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)
	@cp ./bin/terraform-provider-rediscloud_v$(PROVIDER_VERSION) $(PLUGINS_PATH)/$(PLUGINS_PROVIDER_PATH)

website:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), getting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	ln -s ../../../ext/providers/$(PROVIDER_TYPE)/website/$(PROVIDER_TYPE).erb $(GOPATH)/src/$(WEBSITE_REPO)/content/source/layouts/$(PROVIDER_TYPE).erb || true
	ln -s ../../../../ext/providers/$(PROVIDER_TYPE)/website/docs $(GOPATH)/src/$(WEBSITE_REPO)/content/source/docs/providers/$(PROVIDER_TYPE) || true
	@$(MAKE) -C $(GOPATH)/src/$(WEBSITE_REPO) website-provider PROVIDER_PATH=$(shell pwd) PROVIDER_NAME=$(PROVIDER_TYPE)

website-test:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), getting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	ln -s ../../../ext/providers/$(PROVIDER_TYPE)/website/$(PROVIDER_TYPE).erb $(GOPATH)/src/$(WEBSITE_REPO)/content/source/layouts/$(PROVIDER_TYPE).erb || true
	ln -s ../../../../ext/providers/$(PROVIDER_TYPE)/website/docs $(GOPATH)/src/$(WEBSITE_REPO)/content/source/docs/providers/$(PROVIDER_TYPE) || true
	@$(MAKE) -C $(GOPATH)/src/$(WEBSITE_REPO) website-provider-test PROVIDER_PATH=$(shell pwd) PROVIDER_NAME=$(PROVIDER_TYPE)

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test ./internal/provider -v -sweep=ALL $(SWEEPARGS) -timeout 30m
