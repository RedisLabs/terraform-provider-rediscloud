package utils

import (
	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"os"
	"testing"
)

const TestResourcePrefix = "tf-test"

func TestAccRequiresEnvVar(t *testing.T, envVarName string) string {
	envVarValue := os.Getenv(envVarName)
	if envVarValue == "" || envVarValue == "false" {
		t.Skipf("Skipping test because %s is not set.", envVarName)
	}
	return envVarValue
}

func TestAccPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
}

func TestAccAwsPreExistingCloudAccountPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")
}

func TestAccAwsCloudAccountPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_ACCESS_KEY_ID", "AWS_ACCESS_SECRET_KEY", "AWS_CONSOLE_USERNAME", "AWS_CONSOLE_PASSWORD", "AWS_SIGNIN_URL")
}

func TestAccAwsPeeringPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_PEERING_REGION", "AWS_ACCOUNT_ID", "AWS_VPC_ID", "AWS_VPC_CIDR")
}

func TestAccGcpProjectPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "GCP_PROJECT_ID")
}

func TestAccGcpCredentialsPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, "GOOGLE_CREDENTIALS")
}

func TestAccAwsPreExistingTgwCheck(t *testing.T) {
	requireEnvironmentVariables(t, "AWS_TEST_TGW_ID")
}

func requireEnvironmentVariables(t *testing.T, names ...string) {
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}
