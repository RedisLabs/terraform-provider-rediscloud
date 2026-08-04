package provider_test

import (
	"testing"

	provider "github.com/RedisLabs/terraform-provider-rediscloud/provider"
)

func TestProvider(t *testing.T) {
	if err := provider.NewSdkProvider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
