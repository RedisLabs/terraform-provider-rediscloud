package unit_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

func TestCloudOnlyPartialCredentialsKeepExistingDiagnostics(t *testing.T) {
	for attribute, summary := range map[string]string{"api_key": "Missing Redis Cloud API Secret Key", "secret_key": "Missing Redis Cloud API Key"} {
		t.Run(attribute, func(t *testing.T) {
			server, err := factories(t)["rediscloud"]()
			require.NoError(t, err)
			schema, err := server.GetProviderSchema(context.Background(), &tfprotov5.GetProviderSchemaRequest{})
			require.NoError(t, err)
			typ := schema.Provider.ValueType().(tftypes.Object)
			values := map[string]tftypes.Value{}
			for name, valueType := range typ.AttributeTypes {
				values[name] = tftypes.NewValue(valueType, nil)
			}
			values[attribute] = tftypes.NewValue(tftypes.String, "test-credential")
			config, err := tfprotov5.NewDynamicValue(typ, tftypes.NewValue(typ, values))
			require.NoError(t, err)
			resp, err := server.ConfigureProvider(context.Background(), &tfprotov5.ConfigureProviderRequest{TerraformVersion: "1.16.7", Config: &config})
			require.NoError(t, err)
			require.Len(t, resp.Diagnostics, 1)
			require.Equal(t, summary, resp.Diagnostics[0].Summary)
		})
	}
}

func TestUnknownProviderSettingsHaveExplicitDiagnostics(t *testing.T) {
	for _, attribute := range []string{"url", "api_key", "secret_key", "iris_agent_memory_url", "iris_agent_memory_token", "iris_identity_url", "iris_identity_token"} {
		t.Run(attribute, func(t *testing.T) {
			server, err := factories(t)["rediscloud"]()
			require.NoError(t, err)
			schema, err := server.GetProviderSchema(context.Background(), &tfprotov5.GetProviderSchemaRequest{})
			require.NoError(t, err)
			typ := schema.Provider.ValueType().(tftypes.Object)
			values := map[string]tftypes.Value{}
			for name, valueType := range typ.AttributeTypes {
				values[name] = tftypes.NewValue(valueType, nil)
			}
			values["iris_identity_url"] = tftypes.NewValue(tftypes.String, "https://identity.example.com")
			values["iris_identity_token"] = tftypes.NewValue(tftypes.String, "known-token")
			values[attribute] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
			config, err := tfprotov5.NewDynamicValue(typ, tftypes.NewValue(typ, values))
			require.NoError(t, err)
			response, err := server.ConfigureProvider(context.Background(), &tfprotov5.ConfigureProviderRequest{TerraformVersion: "1.16.7", Config: &config})
			require.NoError(t, err)
			require.Len(t, response.Diagnostics, 1)
			require.Equal(t, "Unknown provider setting", response.Diagnostics[0].Summary)
			require.Equal(t, tftypes.NewAttributePath().WithAttributeName(attribute), response.Diagnostics[0].Attribute)
		})
	}
}

func TestUnknownProviderDiagnosticsHaveStableOrder(t *testing.T) {
	server, err := factories(t)["rediscloud"]()
	require.NoError(t, err)
	schema, err := server.GetProviderSchema(context.Background(), &tfprotov5.GetProviderSchemaRequest{})
	require.NoError(t, err)
	typ := schema.Provider.ValueType().(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, valueType := range typ.AttributeTypes {
		values[name] = tftypes.NewValue(valueType, tftypes.UnknownValue)
	}
	config, err := tfprotov5.NewDynamicValue(typ, tftypes.NewValue(typ, values))
	require.NoError(t, err)
	for range 10 {
		response, err := server.ConfigureProvider(context.Background(), &tfprotov5.ConfigureProviderRequest{TerraformVersion: "1.16.7", Config: &config})
		require.NoError(t, err)
		// The mux returns only the first error from a downstream provider.
		require.Len(t, response.Diagnostics, 1)
		require.Equal(t, "Unknown provider setting", response.Diagnostics[0].Summary)
		require.Equal(t, tftypes.NewAttributePath().WithAttributeName("url"), response.Diagnostics[0].Attribute)
	}
}
