package utils

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	sharedTestClient     *client.ApiClient
	sharedTestClientOnce sync.Once
	sharedTestClientErr  error
)

// GetTestClient returns a shared API client for acceptance tests.
// The client is created once and reused across all calls within a test binary.
func GetTestClient() (*client.ApiClient, error) {
	sharedTestClientOnce.Do(func() {
		sharedTestClient, sharedTestClientErr = client.NewClient()
	})
	return sharedTestClient, sharedTestClientErr
}

func SharedTestClient(t *testing.T) *client.ApiClient {
	sharedTestClientOnce.Do(func() {
		sharedTestClient, sharedTestClientErr = client.NewClient()
	})
	if sharedTestClientErr != nil {
		t.Fatalf("Failed to create test API client: %s", sharedTestClientErr)
	}
	return sharedTestClient
}

// CheckNoDatabasesForSubscription verifies that no databases exist in a subscription.
// Combines GetTestClient with CheckNoDatabasesInSubscription for convenience in test check functions.
func CheckNoDatabasesForSubscription(ctx context.Context, subId int) error {
	api, err := GetTestClient()
	if err != nil {
		return err
	}
	return CheckNoDatabasesInSubscription(ctx, subId, api)
}

func GetTestConfig(t *testing.T, testFile string) string {
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	return string(content)
}

// TestResourcePrefix returns the prefix used for all test resource names.
// Reads TEST_RESOURCE_PREFIX env var; defaults to "tf-test".
func TestResourcePrefix() string {
	if prefix := os.Getenv("TEST_RESOURCE_PREFIX"); prefix != "" {
		return prefix
	}
	return "tf-test"
}

// CheckNoDatabasesInSubscription verifies that no databases exist in a subscription.
// Used by CMK tests to confirm creation-plan databases were cleaned up after activation.
func CheckNoDatabasesInSubscription(ctx context.Context, subId int, api *client.ApiClient) error {
	dbList := api.Client.Database.List(ctx, subId)
	var dbIds []int
	for dbList.Next() {
		dbIds = append(dbIds, *dbList.Value().ID)
	}
	if dbList.Err() != nil {
		return fmt.Errorf("failed to list databases in subscription %d: %w", subId, dbList.Err())
	}
	if len(dbIds) > 0 {
		return fmt.Errorf("expected no databases in subscription %d, but found %d: %v", subId, len(dbIds), dbIds)
	}
	return nil
}

// RandomWithPrefix generates a unique name with the test resource prefix
// and a random suffix. Defaults to 6-char suffix to stay within the API's
// 40-char name limit; pass an explicit length to override.
func RandomWithPrefix(n ...int) string {
	length := 6
	if len(n) > 0 && n[0] > 0 {
		length = n[0]
	}
	return TestResourcePrefix() + "-" + acctest.RandString(length)
}
