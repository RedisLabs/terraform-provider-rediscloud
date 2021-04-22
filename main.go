package main

import (
	"context"
	"flag"
	"github.com/RedisLabs/terraform-provider-rediscloud/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"log"
)

var (
	// Provided by goreleaser configuration for each binary
	// Allows goreleaser to pass version details
	version string = "dev"

	// Allows goreleaser pass the specific commit details
	commit string = ""
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{ProviderFunc: provider.New(version)}

	if debugMode {
		err := plugin.Debug(context.Background(), "RedisLabs/rediscloud", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}
