package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func MuxProviderServerCreator(sdkProvider *schema.Provider) (func() tfprotov5.ProviderServer, error) {
	ctx := context.Background()
	providers := []func() tfprotov5.ProviderServer{
		sdkProvider.GRPCProvider,
		// terraform-framework provider will go here (take from arguments)
		// providerserver.NewProtocol5(fwProvider)
	}

	muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)

	if err != nil {
		return nil, err
	}

	return muxServer.ProviderServer, nil
}
