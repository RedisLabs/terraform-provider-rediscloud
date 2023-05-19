package main

import (
	"flag"
	"log"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

var (
	// Provided by goreleaser configuration for each binary
	// Allows goreleaser to pass version details
	version = "dev"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{ProviderFunc: provider.New(version)}

	// Prevent logger from prepending date/time to logs, which breaks log-level parsing/filtering
	log.SetFlags(0)

	if debugMode {
		opts.Debug = true
		opts.ProviderAddr = "RedisLabs/rediscloud"
	}

	plugin.Serve(opts)
}
