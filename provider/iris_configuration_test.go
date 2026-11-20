package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIrisProviderConfiguration(t *testing.T) {
	for _, setting := range irisSettings {
		t.Setenv(setting.env, "")
	}
	t.Setenv("REDISCLOUD_IRIS_IDENTITY_URL", "https://identity.example.com/prefix")
	t.Setenv("REDISCLOUD_IRIS_IDENTITY_TOKEN", "environment-token")
	services, err := irisClients(map[string]string{})
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.NotNil(t, services["identity"])
	services, err = irisClients(map[string]string{"iris_agent_memory_url": "https://memory.example.com", "iris_agent_memory_token": "other-token"})
	require.NoError(t, err)
	require.Len(t, services, 2)
	_, err = irisClients(map[string]string{"iris_agent_memory_token": "only-token"})
	require.ErrorContains(t, err, "iris_agent_memory_url")
	_, err = irisClients(map[string]string{"iris_agent_memory_url": "https://memory.example.com"})
	require.ErrorContains(t, err, "bearer token")
}
