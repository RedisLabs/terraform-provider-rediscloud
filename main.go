package main

import (
	"flag"
	"log"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
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

	// Create a muxed provider server that combines SDK v2 and Plugin Framework providers
	muxServer, err := provider.MuxProviderServerCreator(
		provider.NewSdkProvider(version)(),
		provider.NewFrameworkProvider(version)(),
	)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf5server.ServeOpt

	// Prevent logger from prepending date/time to logs, which breaks log-level parsing/filtering
	log.SetFlags(0)

	if debugMode {
		serveOpts = append(serveOpts, tf5server.WithManagedDebug())
	}

	err = tf5server.Serve("RedisLabs/rediscloud", muxServer, serveOpts...)
	if err != nil {
		log.Fatal(err)
	}
}
