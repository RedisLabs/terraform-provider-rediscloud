package envchecks

import (
	"os"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

// RequireEnvVars fails the test if any of the named environment
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

func GCPProviderCheck(t *testing.T) {
	t.Helper()
	RequireEnvVars(t, "GOOGLE_CREDENTIALS", "GCP_PROJECT_ID")
}

func ValueAndCheck(t *testing.T, key string) (string, func()) {
	t.Helper()
	value := os.Getenv(key)
	return value, func() { RequireEnvVars(t, key) }
}

func AWSBYOCValueAndCheck(t *testing.T) (string, func()) {
	t.Helper()
	return ValueAndCheck(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")
}

func GCPProjectValueAndCheck(t *testing.T) (string, func()) {
	t.Helper()
	return ValueAndCheck(t, "GCP_PROJECT_ID")
}

type AWSPeering struct {
	Region    string
	AccountId string
	VpcId     string
	VpcCidr   string
}

func AwsPeeringValueAndCheck(t *testing.T) (AWSPeering, func()) {
	t.Helper()
	region, regionCheck := ValueAndCheck(t, "AWS_PEERING_REGION")
	accountId, accountIdCheck := ValueAndCheck(t, "AWS_ACCOUNT_ID")
	vpcId, vpcIdCheck := ValueAndCheck(t, "AWS_VPC_ID")
	vpcCidr, vpcCidrCheck := ValueAndCheck(t, "AWS_VPC_CIDR")
	return AWSPeering{Region: region, AccountId: accountId, VpcId: vpcId, VpcCidr: vpcCidr}, func() { regionCheck(); accountIdCheck(); vpcIdCheck(); vpcCidrCheck() }
}
