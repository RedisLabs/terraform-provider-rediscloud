default: testacc

plugin_version = 99.99.99
plugins_mirror_path = ~/.terraform.d/plugins
plugins_provider_path = registry.terraform.io/RedisLabs/rediscloud/$(plugin_version)/darwin_amd64/

WEBSITE_REPO=github.com/hashicorp/terraform-website
PROVIDER_TYPE=rediscloud

.PHONY: build clean testacc install_local website website-test

build:
	@echo "Building local provider binary"
	@mkdir -p ./bin
	go build -o bin/terraform-provider-rediscloud_v$(plugin_version)

clean:
	@echo "Deleting local provider binary"
	rm -rf ./bin

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

install_local: build
	@echo "Installing local provider binary to plugins mirror path $(plugins_mirror_path)/$(plugins_provider_path)"
	@mkdir -p $(plugins_mirror_path)/$(plugins_provider_path)
	@cp ./bin/terraform-provider-rediscloud_v$(plugin_version) $(plugins_mirror_path)/$(plugins_provider_path)

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
