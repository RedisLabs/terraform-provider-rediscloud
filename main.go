package main

import (
	"github.com/RedisLabs/terraform-provider-rediscloud/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

var (
	// Provided by goreleaser configuration for each binary
	// Allows goreleaser to pass version details
	version string = "dev"

	// Allows goreleaser pass the specific commit details
	commit string = ""
)

func main() {
	plugin.Serve(&plugin.ServeOpts{ProviderFunc: provider.New(version)})
}
