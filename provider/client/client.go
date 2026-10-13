package client

import (
	"os"
	"sync"
	"testing"
	"time"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

type ApiClient struct {
	Client *rediscloudApi.Client

	// WaitDelayOverride and WaitPollIntervalOverride replace the corresponding resource-state waiter
	// timing when non-zero. Production leaves both zero, so each waiter keeps the literal timing at its
	// own call site — those timings differ per waiter and are not interchangeable, so there is
	// deliberately no shared default here to flatten them.
	//
	// Tests driving an in-memory API set them to something tiny: the fixture answers on the first poll,
	// so retry.StateChangeConf.Delay — an unconditional sleep before that first poll — is pure dead time.
	// Carried on the client rather than in package-level state so each test owns its own values, which
	// keeps parallel tests race-free.
	//
	// Zero means "not overridden", so a literal zero cannot be requested. That is not a gap worth
	// closing with a pointer: use a millisecond. Zero would be marginally faster still — it drops the
	// pre-poll sleep and puts retry on its exponential backoff rather than a fixed interval — but the
	// delay is paid once per waiter and the fixture responds immediately, so the difference is
	// unmeasurable.
	WaitDelayOverride        time.Duration
	WaitPollIntervalOverride time.Duration
}

// WaitDelay returns def, unless a test has overridden the waiter delay.
func (c *ApiClient) WaitDelay(def time.Duration) time.Duration {
	if c.WaitDelayOverride > 0 {
		return c.WaitDelayOverride
	}
	return def
}

// WaitPollInterval returns def, unless a test has overridden the waiter poll interval.
func (c *ApiClient) WaitPollInterval(def time.Duration) time.Duration {
	if c.WaitPollIntervalOverride > 0 {
		return c.WaitPollIntervalOverride
	}
	return def
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

// MustTestClient returns an API client or logs an error to the test logs and fails the test instantly
func MustTestClient(t *testing.T) *ApiClient {
	sharedClient, sharedClientErr = GetTestClient()
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
