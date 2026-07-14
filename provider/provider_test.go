package provider_test

import (
	"testing"

	provider "github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
)

func TestProvider(t *testing.T) {
	if err := provider.NewSdkProvider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccAwsCloudAccountPreCheck(t *testing.T) bool {
	return envchecks.RequireEnvVars(t, "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_CONSOLE_USERNAME", "AWS_CONSOLE_PASSWORD", "AWS_SIGNIN_URL")
}
