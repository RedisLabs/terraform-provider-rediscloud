default: testacc

plugin_version = 99.99.99
plugins_mirror_path = ~/Library/Application\ Support/io.terraform/plugins
plugins_provider_path = registry.redislabs.com/redislabs/rediscloud/$(plugin_version)/darwin_amd64/

WEBSITE_REPO=github.com/hashicorp/terraform-website
PROVIDER_TYPE=rediscloud

# Run acceptance tests
.PHONY: testacc install_macos website website-test

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

install_macos:
	go build -o dist/terraform-provider-rediscloud_v$(plugin_version)

	mkdir -p $(plugins_mirror_path)/$(plugins_provider_path)
	cp dist/terraform-provider-rediscloud_v$(plugin_version) $(plugins_mirror_path)/$(plugins_provider_path)

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

build_0_13: fmtcheck
	@mkdir -p $(PROVIDER_PATH)
	go build -o $(PROVIDER_PATH)/terraform-provider-$(PROVIDER_NAMESPACE)_v$(PROVIDER_VERSION)

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test ./internal/provider -v -sweep=ALL $(SWEEPARGS) -timeout 30m
