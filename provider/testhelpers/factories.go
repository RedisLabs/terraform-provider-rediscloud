package testhelpers

import (
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
)

// ProtoV5ProviderFactories returns a fresh provider factories map for use in
// acceptance tests. A new map is returned each call to prevent cross-test mutation.
func ProtoV5ProviderFactories() map[string]func() (tfprotov5.ProviderServer, error) {
	return map[string]func() (tfprotov5.ProviderServer, error){
		"rediscloud": func() (tfprotov5.ProviderServer, error) {
			muxServer, err := provider.MuxProviderServerCreator(
				provider.NewSdkProvider("99.99.99")(),
				provider.NewFrameworkProvider("99.99.99")(),
			)
			if err != nil {
				return nil, err
			}
			return muxServer(), nil
		},
	}
}
