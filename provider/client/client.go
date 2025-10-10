package client

import (
	"os"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

type ApiClient struct {
	Client *rediscloudApi.Client
}

// NewClient creates a new ApiClient using environment variables for configuration.
// This is useful for tests that need to create a client before the provider is configured.
func NewClient() (*ApiClient, error) {
	var config []rediscloudApi.Option

	url := os.Getenv("REDISCLOUD_URL")
	if url != "" {
		config = append(config, rediscloudApi.BaseURL(url))
	}

	client, err := rediscloudApi.NewClient(config...)
	if err != nil {
		return nil, err
	}

	return &ApiClient{
		Client: client,
	}, nil
}
