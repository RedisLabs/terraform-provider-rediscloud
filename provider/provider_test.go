package provider_test

import (
	"sync"
	"testing"

	provider "github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
)

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
	if err := provider.NewSdkProvider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// testAccAwsApiCredsPreCheck requires only the AWS API credentials needed by the
// hashicorp/aws external provider for tests that provision AWS resources directly
// (e.g. the AWS CMK tests, which create KMS keys + key policies in-fixture).
func testAccAwsApiCredsPreCheck(t *testing.T) {
	envchecks.RequireEnvVars(t, "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY")
}

func testAccAwsCloudAccountPreCheck(t *testing.T) {
	envchecks.RequireEnvVars(t, "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_CONSOLE_USERNAME", "AWS_CONSOLE_PASSWORD", "AWS_SIGNIN_URL")
}

func testAccAwsPeeringPreCheck(t *testing.T) {
	envchecks.RequireEnvVars(t, "AWS_PEERING_REGION", "AWS_ACCOUNT_ID", "AWS_VPC_ID", "AWS_VPC_CIDR")
}

func testAccGcpProjectPreCheck(t *testing.T) {
	envchecks.RequireEnvVars(t, "GCP_PROJECT_ID")
}

func testAccGcpCredentialsPreCheck(t *testing.T) {
	envchecks.RequireEnvVars(t, "GOOGLE_CREDENTIALS")
}

func testAccAwsCredentialsPreCheck(t *testing.T) {
	envchecks.RequireEnvVars(t, "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION")
}
