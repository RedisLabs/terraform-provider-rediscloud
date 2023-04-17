package main

import (
	"context"
	"flag"
	"log"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
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
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{ProviderFunc: provider.New(version)}

	// Prevent logger from prepending date/time to logs, which breaks log-level parsing/filtering
	log.SetFlags(0)

	if debugMode {
		err := plugin.Debug(context.Background(), "RedisLabs/rediscloud", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}
