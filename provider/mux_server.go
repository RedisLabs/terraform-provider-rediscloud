package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// MuxProviderServerCreator creates a muxed provider server that combines the SDK v2 provider
// with the Plugin Framework provider, allowing resources to be served from either implementation.
func MuxProviderServerCreator(sdkProvider *schema.Provider, frameworkProvider provider.Provider) (func() tfprotov5.ProviderServer, error) {
	ctx := context.Background()
	providers := []func() tfprotov5.ProviderServer{
		sdkProvider.GRPCProvider,
		providerserver.NewProtocol5(frameworkProvider),
	}

	muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		return nil, err
	}

	return muxServer.ProviderServer, nil
}
