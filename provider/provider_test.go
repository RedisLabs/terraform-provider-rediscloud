package provider

import (
	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var sdkProvider *schema.Provider
var frameworkProvider provider.Provider

var providerFactories map[string]func() (tfprotov5.ProviderServer, error)

func init() {
	sdkProvider = NewSdkProvider("dev")()
	frameworkProvider = NewFrameworkProvider("dev")()
	providerFactories = map[string]func() (tfprotov5.ProviderServer, error){
		"rediscloud": func() (tfprotov5.ProviderServer, error) {
			// This method signature will include frameworkProvider
			muxProviderServerCreator, err := MuxProviderServerCreator(sdkProvider, frameworkProvider)
			if err != nil {
				return nil, err
			}
			return muxProviderServerCreator(), nil
		},
	}
}

func TestSdkProvider(t *testing.T) {
	if err := NewSdkProvider("dev")().InternalValidate(); err != nil {
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

func requireEnvironmentVariables(t *testing.T, names ...string) {
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}
