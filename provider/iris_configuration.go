package provider

import (
	"fmt"
	"os"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

type irisSetting struct {
	name, env, description string
	sensitive              bool
}

var irisSettings = []irisSetting{
	{"iris_agent_memory_url", "REDISCLOUD_IRIS_AGENT_MEMORY_URL", "Direct Iris Agent Memory control-plane origin or path prefix.", false},
	{"iris_agent_memory_token", "REDISCLOUD_IRIS_AGENT_MEMORY_TOKEN", "Bearer token for the direct Iris Agent Memory control plane.", true},
	{"iris_identity_url", "REDISCLOUD_IRIS_IDENTITY_URL", "Direct Iris Identity Service control-plane origin or path prefix.", false},
	{"iris_identity_token", "REDISCLOUD_IRIS_IDENTITY_TOKEN", "Bearer token for the direct Iris Identity Service control plane.", true},
}

// Both halves of the mux use identical settings and environment fallbacks.
func irisClients(values map[string]string) (map[string]*client.IrisServiceClient, error) {
	for _, setting := range irisSettings {
		if values[setting.name] == "" {
			values[setting.name] = os.Getenv(setting.env)
		}
	}
	for _, key := range []struct{ name, env string }{{"api_key", "REDISCLOUD_ACCESS_KEY"}, {"secret_key", "REDISCLOUD_SECRET_KEY"}} {
		if values[key.name] == "" {
			values[key.name] = os.Getenv(key.env)
		}
	}
	services := map[string]*client.IrisServiceClient{}
	for _, service := range []struct{ name, prefix string }{{"agent-memory", "iris_agent_memory"}, {"identity", "iris_identity"}} {
		endpoint, token := values[service.prefix+"_url"], values[service.prefix+"_token"]
		if endpoint == "" && token == "" {
			continue
		}
		if endpoint == "" {
			return nil, fmt.Errorf("%s_url is required when its token is set", service.prefix)
		}
		serviceConfig := client.IrisServiceConfig{URL: endpoint, Token: token}
		if service.name == "agent-memory" {
			serviceConfig.APIKey = values["api_key"]
			serviceConfig.SecretKey = values["secret_key"]
		}
		connector, err := client.NewIrisServiceClient(serviceConfig)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", service.prefix, err)
		}
		services[service.name] = connector
	}
	return services, nil
}
