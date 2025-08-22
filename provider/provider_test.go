package provider

import (
	"os"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const testResourcePrefix = "tf-test"

var testProvider *schema.Provider
var providerFactories map[string]func() (*schema.Provider, error)

func init() {
	testProvider = New("dev")()
	providerFactories = map[string]func() (*schema.Provider, error){
		"rediscloud": func() (*schema.Provider, error) {
			return testProvider, nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, utils.RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
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

func requireEnvironmentVariables(t *testing.T, names ...string) {
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}

func testAccRequiresEnvVar(t *testing.T, envVarName string) string {
	envVarValue := os.Getenv(envVarName)
	if envVarValue == "" || envVarValue == "false" {
		t.Skipf("Skipping test because %s is not set.", envVarName)
	}
	return envVarValue
}
