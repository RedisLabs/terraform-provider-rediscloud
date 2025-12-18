package provider

import (
	"os"
	"sync"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var providerFactories map[string]func() (*schema.Provider, error)

func init() {
	// Create a fresh provider instance for each test
	providerFactories = map[string]func() (*schema.Provider, error){
		"rediscloud": func() (*schema.Provider, error) {
			return New("dev")(), nil
		},
	}
}

// sharedTestClient returns an API client for use in test check functions.
// The client is lazily initialised and shared across all tests.
var (
	sharedClient     *client.ApiClient
	sharedClientOnce sync.Once
	sharedClientErr  error
)

func sharedTestClient(t *testing.T) *client.ApiClient {
	sharedClientOnce.Do(func() {
		sharedClient, sharedClientErr = client.NewClient()
	})
	if sharedClientErr != nil {
		t.Fatalf("Failed to create test API client: %s", sharedClientErr)
	}
	return sharedClient
}

// getTestClient returns an API client for use in CheckDestroy and other
// functions that don't have access to *testing.T. Returns an error if the
// client cannot be created.
func getTestClient() (*client.ApiClient, error) {
	sharedClientOnce.Do(func() {
		sharedClient, sharedClientErr = client.NewClient()
	})
	return sharedClient, sharedClientErr
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
}

func testAccAwsPreExistingCloudAccountPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")
}

func testAccAwsCloudAccountPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_ACCESS_KEY_ID", "AWS_ACCESS_SECRET_KEY", "AWS_CONSOLE_USERNAME", "AWS_CONSOLE_PASSWORD", "AWS_SIGNIN_URL")
}

func testAccAwsPeeringPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_PEERING_REGION", "AWS_ACCOUNT_ID", "AWS_VPC_ID", "AWS_VPC_CIDR")
}

func testAccGcpProjectPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "GCP_PROJECT_ID")
}

func testAccGcpCredentialsPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "GOOGLE_CREDENTIALS")
}

func testAccAwsPreExistingTgwCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_TEST_TGW_ID")
}

func testAccAwsCredentialsPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION")
}

func testAccRedisCloudAwsAccountPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "REDISCLOUD_AWS_ACCOUNT_ID")
}

func requireEnvironmentVariables(t *testing.T, names ...string) {
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}
