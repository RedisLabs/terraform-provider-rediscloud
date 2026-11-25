package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"

	"github.com/stretchr/testify/require"
)

func TestIrisProviderConfiguration(t *testing.T) {
	t.Setenv("REDISCLOUD_ACCESS_KEY", "")
	t.Setenv("REDISCLOUD_SECRET_KEY", "")
	for _, setting := range irisSettings {
		t.Setenv(setting.env, "")
	}
	t.Setenv("REDISCLOUD_IRIS_IDENTITY_URL", "https://identity.example.com/prefix")
	t.Setenv("REDISCLOUD_IRIS_IDENTITY_TOKEN", "environment-token")
	services, err := irisClients(map[string]string{}, "test")
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.NotNil(t, services["identity"])
	services, err = irisClients(map[string]string{"iris_agent_memory_url": "https://memory.example.com", "iris_agent_memory_token": "other-token"}, "test")
	require.NoError(t, err)
	require.Len(t, services, 2)
	_, err = irisClients(map[string]string{"iris_agent_memory_token": "only-token"}, "test")
	require.ErrorContains(t, err, "iris_agent_memory_url")
	_, err = irisClients(map[string]string{"iris_agent_memory_url": "https://memory.example.com"}, "test")
	require.ErrorContains(t, err, "bearer token")
}

func TestIrisRejectsIncompleteCloudCredentials(t *testing.T) {
	t.Setenv("REDISCLOUD_ACCESS_KEY", "")
	t.Setenv("REDISCLOUD_SECRET_KEY", "")
	for _, setting := range irisSettings {
		t.Setenv(setting.env, "")
	}
	for _, credentials := range []map[string]string{{"api_key": "key"}, {"secret_key": "secret"}} {
		credentials["iris_identity_url"] = "https://identity.example.com"
		credentials["iris_identity_token"] = "bearer"
		_, err := irisClients(credentials, "test")
		require.ErrorContains(t, err, "api_key and secret_key must be provided together")
	}
	_, err := irisClients(map[string]string{"iris_identity_url": "https://identity.example.com", "iris_identity_token": "bearer"}, "test")
	require.NoError(t, err)
}

func TestSDKValidatesIrisWithoutOwningItsClients(t *testing.T) {
	t.Setenv("REDISCLOUD_ACCESS_KEY", "")
	t.Setenv("REDISCLOUD_SECRET_KEY", "")
	for _, setting := range irisSettings {
		t.Setenv(setting.env, "")
	}
	p := NewSdkProvider("test")()
	data := schema.TestResourceDataRaw(t, p.Schema, map[string]any{"iris_identity_url": "https://identity.example.com", "iris_identity_token": "bearer"})
	configured, d := p.ConfigureContextFunc(context.Background(), data)
	require.False(t, d.HasError(), d)
	require.Nil(t, configured.(*client.ApiClient).Iris)
	bad := schema.TestResourceDataRaw(t, p.Schema, map[string]any{"iris_identity_url": "http://identity.example.com", "iris_identity_token": "bearer"})
	_, d = p.ConfigureContextFunc(context.Background(), bad)
	require.True(t, d.HasError())
}

func TestIrisClientsIncludeProviderVersion(t *testing.T) {
	for _, setting := range irisSettings {
		t.Setenv(setting.env, "")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "terraform-provider-rediscloud/1.2.3 (iris)", req.UserAgent())
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	services, err := irisClients(map[string]string{"iris_identity_url": server.URL, "iris_identity_token": "test-token", "api_key": "test-key", "secret_key": "test-secret"}, "1.2.3")
	require.NoError(t, err)
	require.NoError(t, services["identity"].Do(context.Background(), http.MethodGet, "/v1/api-keys", nil, nil, nil))
}
