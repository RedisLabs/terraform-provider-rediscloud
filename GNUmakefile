default: testacc

plugin_version = 99.99.99
plugins_mirror_path = ~/Library/Application\ Support/io.terraform/plugins
plugins_provider_path = registry.redislabs.com/redislabs/rediscloud/$(plugin_version)/darwin_amd64/

# Run acceptance tests
.PHONY: testacc install_macos

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

install_macos:
	go build -o dist/terraform-provider-rediscloud_v$(plugin_version)

	mkdir -p $(plugins_mirror_path)/$(plugins_provider_path)
	cp dist/terraform-provider-rediscloud_v$(plugin_version) $(plugins_mirror_path)/$(plugins_provider_path)
