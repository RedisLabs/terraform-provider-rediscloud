package envchecks

import (
	"os"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

// RequireEnvVars fails the test if any of the named environment
// variables are not set.
func RequireEnvVars(t *testing.T, names ...string) bool {
	t.Helper()
	missingEnvVars := false
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			t.Errorf("Missing `%s` environment variable", name)
			missingEnvVars = true
		}
	}
	return !missingEnvVars
}

// RedisCloudCheck checks that the minimum provider configuration (URL, access key
// and secret key) are present. Use RequireEnvVars directly when additional
// variables are needed.
func RedisCloudCheck(t *testing.T) bool {
	t.Helper()
	return RequireEnvVars(t, rediscloudApi.RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
}

func AWSProviderCheck(t *testing.T) bool {
	t.Helper()
	return RequireEnvVars(t, "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY")
}

func GCPProviderCheck(t *testing.T) bool {
	t.Helper()
	return RequireEnvVars(t, "GOOGLE_CREDENTIALS", "GCP_PROJECT_ID")
}

func ValueAndCheck(key string) (string, func(t *testing.T) bool) {
	value := os.Getenv(key)
	check := func(t *testing.T) bool {
		return RequireEnvVars(t, key)
	}
	return value, check
}

func AWSBYOCValueAndCheck() (string, func(t *testing.T) bool) {
	return ValueAndCheck("AWS_TEST_CLOUD_ACCOUNT_NAME")
}

func GCPProjectValueAndCheck() (string, func(t *testing.T) bool) {
	return ValueAndCheck("GCP_PROJECT_ID")
}

type AWSPeering struct {
	Region    string
	AccountId string
	VpcId     string
	VpcCidr   string
}

func AwsPeeringValueAndCheck() (AWSPeering, func(t *testing.T) bool) {
	region, regionCheck := ValueAndCheck("AWS_PEERING_REGION")
	accountId, accountIdCheck := ValueAndCheck("AWS_ACCOUNT_ID")
	vpcId, vpcIdCheck := ValueAndCheck("AWS_VPC_ID")
	vpcCidr, vpcCidrCheck := ValueAndCheck("AWS_VPC_CIDR")
	check := func(t *testing.T) bool {
		return composeChecks(t, regionCheck, accountIdCheck, vpcIdCheck, vpcCidrCheck)
	}
	return AWSPeering{Region: region, AccountId: accountId, VpcId: vpcId, VpcCidr: vpcCidr}, check
}

// ComposePreChecks should be used to define all PreCheck functions for testing, allowing all the individual pre check functions to be combined and succeed or fail with a full list of missing env vars
func ComposePreChecks(t *testing.T, checks ...func(t *testing.T) bool) func() {
	t.Helper()
	return func() {
		t.Helper()
		if !composeChecks(t, checks...) {
			t.FailNow()
		}
	}
}

// composeChecks allows us to internally compose multiple check functions into a single one
func composeChecks(t *testing.T, checks ...func(t *testing.T) bool) bool {
	t.Helper()
	passed := true
	for _, check := range checks {
		if !check(t) {
			passed = false
		}
	}
	return passed
}
