package cloudaccount_test

import (
	"os"
	"regexp"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func testAccPreCheck(t *testing.T) {
	t.Helper()
	testhelpers.RequireEnvVars(t, provider.RedisCloudUrlEnvVar, rediscloudapi.AccessKeyEnvVar, rediscloudapi.SecretKeyEnvVar, "AWS_TEST_CLOUD_ACCOUNT_NAME")
}

func TestAccDataSourceRedisCloudCloudAccount_basic(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	const testCloudAccount = "data.rediscloud_cloud_account.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
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
