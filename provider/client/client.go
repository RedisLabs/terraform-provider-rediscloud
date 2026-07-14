package client

import (
	"os"
	"sync"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

type ApiClient struct {
	Client *rediscloudApi.Client
}

var (
	sharedClient     *ApiClient
	sharedClientOnce sync.Once
	sharedClientErr  error
)

// NewClient creates a new ApiClient using environment variables for configuration.
// This is useful for tests that need to create a client before the provider is configured.
func NewClient() (*ApiClient, error) {
	var config []rediscloudApi.Option

	url := os.Getenv(rediscloudApi.RedisCloudUrlEnvVar)
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

func SharedTestClient(t *testing.T) *ApiClient {
	sharedClientOnce.Do(func() {
		sharedClient, sharedClientErr = NewClient()
	})
	if sharedClientErr != nil {
		t.Fatalf("Failed to create test API client: %s", sharedClientErr)
	}
	return sharedClient
}

// GetTestClient returns an API client for use in CheckDestroy and other
// functions that don't have access to *testing.T. Returns an error if the
// client cannot be created.
func GetTestClient() (*ApiClient, error) {
	sharedClientOnce.Do(func() {
		sharedClient, sharedClientErr = NewClient()
	})
	return sharedClient, sharedClientErr
}
