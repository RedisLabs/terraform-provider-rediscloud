package cloudaccount_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func testAccPreCheck(t *testing.T) {
	t.Helper()
	testhelpers.BasicPreCheck(t)
	testhelpers.RequireEnvVars(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")
}

func TestAccDataSourceRedisCloudCloudAccount_basic(t *testing.T) {

	name := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	const testCloudAccount = "data.rediscloud_cloud_account.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // test doesn't create a resource at the moment, so don't need to check anything
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/datasource_basic.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(name),
				},
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
