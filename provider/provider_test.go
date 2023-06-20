package provider

import (
	rediscloud_api "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var sdkv2Provider *schema.Provider

// frameworkProvider := NewFWProvider()
var providerFactories map[string]func() (tfprotov5.ProviderServer, error)

func init() {
	sdkv2Provider = New("dev")()
	providerFactories = map[string]func() (tfprotov5.ProviderServer, error){
		"rediscloud": func() (tfprotov5.ProviderServer, error) {
			// This method signature will include frameworkProvider
			muxProviderServerCreator, err := MuxProviderServerCreator(sdkv2Provider)
			if err != nil {
				return nil, err
			}
			return muxProviderServerCreator(), nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	requireEnvironmentVariables(t, RedisCloudUrlEnvVar, rediscloud_api.AccessKeyEnvVar, rediscloud_api.SecretKeyEnvVar)
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

func requireEnvironmentVariables(t *testing.T, names ...string) {
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}
