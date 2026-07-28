package provider_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"

	provider "github.com/RedisLabs/terraform-provider-rediscloud/provider"
)

func TestProvider(t *testing.T) {
	if err := provider.NewSdkProvider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestFrameworkProvider(t *testing.T) {
	server := providerserver.NewProtocol5(provider.NewFrameworkProvider("99.99.99")())()
	assertProviderSchema(t, server)
}

func TestMuxServer(t *testing.T) {
	muxServer, err := provider.MuxProviderServerCreator(
		provider.NewSdkProvider("99.99.99")(),
		provider.NewFrameworkProvider("99.99.99")(),
	)
	if err != nil {
		t.Fatalf("failed to create muxed provider server: %s", err)
	}
	assertProviderSchema(t, muxServer())
}

func assertProviderSchema(t *testing.T, server tfprotov5.ProviderServer) {
	t.Helper()
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov5.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema failed: %s", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov5.DiagnosticSeverityError {
			t.Fatalf("provider schema error: %s: %s", d.Summary, d.Detail)
		}
	}
}
