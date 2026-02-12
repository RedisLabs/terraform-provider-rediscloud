package cloudaccount_test

import (
	"os"
	"regexp"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

var protoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"rediscloud": func() (tfprotov5.ProviderServer, error) {
		muxServer, err := provider.MuxProviderServerCreator(
			provider.NewSdkProvider("dev")(),
			provider.NewFrameworkProvider("dev")(),
		)
		if err != nil {
			return nil, err
		}
		return muxServer(), nil
	},
}

func testAccPreCheck(t *testing.T) {
	for _, name := range []string{provider.RedisCloudUrlEnvVar, rediscloudapi.AccessKeyEnvVar, rediscloudapi.SecretKeyEnvVar, "AWS_TEST_CLOUD_ACCOUNT_NAME"} {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}

func TestAccDataSourceRedisCloudCloudAccount_basic(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	const testCloudAccount = "data.rediscloud_cloud_account.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             nil, // test doesn't create a resource at the moment, so don't need to check anything
		Steps: []resource.TestStep{
			{
				Config: utils.RenderTestConfig(t, "./testdata/datasource_basic.tf", map[string]string{
					"__NAME__": name,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(testCloudAccount, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testCloudAccount, "provider_type", "AWS"),
					resource.TestCheckResourceAttr(testCloudAccount, "name", name),
					resource.TestCheckResourceAttrSet(testCloudAccount, "access_key_id"),
				),
			},
		},
	})
}
