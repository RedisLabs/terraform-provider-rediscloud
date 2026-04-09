package utils

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func AccRequiresEnvVar(t *testing.T, envVarName string) string {
	envVarValue := os.Getenv(envVarName)
	if envVarValue == "" || envVarValue == "false" {
		t.Skipf("Skipping test because %s is not set.", envVarName)
	}
	return envVarValue
}

func GetTestConfig(t *testing.T, testFile string) string {
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	return string(content)
}

// RenderTestConfig loads a test fixture and replaces named placeholders with values.
// Placeholders in fixtures use the format __PLACEHOLDER_NAME__ (e.g., __CLOUD_ACCOUNT__).
// This allows fixtures to be valid HCL that can be formatted with terraform fmt.
//
// For booleans, use "__PLACEHOLDER__" == "true" pattern in fixtures and pass "true"/"false" strings.
// For numbers, use tonumber("__PLACEHOLDER__") pattern in fixtures and pass number as string.
func RenderTestConfig(t *testing.T, testFile string, vars map[string]string) string {
	config := GetTestConfig(t, testFile)
	for placeholder, value := range vars {
		config = strings.ReplaceAll(config, placeholder, value)
	}
	return config
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
