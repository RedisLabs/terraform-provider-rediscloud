package envchecks

import (
	"os"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

// RequireEnvVars skips or fails the test if any of the named environment
// variables are not set.
func RequireEnvVars(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}

// RedisCloudCheck checks that the minimum provider configuration (URL, access key
// and secret key) are present. Use RequireEnvVars directly when additional
// variables are needed.
func RedisCloudCheck(t *testing.T) {
	t.Helper()
	RequireEnvVars(t, rediscloudApi.RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
}

func AWSProviderCheck(t *testing.T) {
	t.Helper()
	RequireEnvVars(t, "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY")
}

func AWSProviderAndRegionCheck(t *testing.T) {
	t.Helper()
	AWSProviderCheck(t)
	RequireEnvVars(t, "AWS_REGION")
}

func AWSBYOCloudAccountCheck(t *testing.T) {
	t.Helper()
	RequireEnvVars(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")
}

func AwsPeeringCheck(t *testing.T) {
	t.Helper()
	RequireEnvVars(t, "AWS_PEERING_REGION", "AWS_ACCOUNT_ID", "AWS_VPC_ID", "AWS_VPC_CIDR")
}

func GCPProjectCheck(t *testing.T) {
	t.Helper()
	RequireEnvVars(t, "GCP_PROJECT_ID")
}

func GCPProviderCheck(t *testing.T) {
	t.Helper()
	GCPProjectCheck(t)
	RequireEnvVars(t, "GOOGLE_CREDENTIALS")
}
