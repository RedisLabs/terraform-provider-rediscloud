package provider_test

import (
	"os"
	"sync"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"

	providerpkg "github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var protoV5ProviderFactories map[string]func() (tfprotov5.ProviderServer, error)

func init() {
	protoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
		"rediscloud": func() (tfprotov5.ProviderServer, error) {
			muxServer, err := providerpkg.MuxProviderServerCreator(
				providerpkg.NewSdkProvider("99.99.99")(),
				providerpkg.NewFrameworkProvider("99.99.99")(),
			)
			if err != nil {
				return nil, err
			}
			return muxServer(), nil
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
	if err := providerpkg.NewSdkProvider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, providerpkg.RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
}

func testAccAwsPreExistingCloudAccountPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")
}

// testAccAwsApiCredsPreCheck requires only the AWS API credentials needed by the
// hashicorp/aws external provider for tests that provision AWS resources directly
// (e.g. the AWS CMK tests, which create KMS keys + key policies in-fixture).
func testAccAwsApiCredsPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY")
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

func testAccAwsCredentialsPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION")
}

func requireEnvironmentVariables(t *testing.T, names ...string) {
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}
