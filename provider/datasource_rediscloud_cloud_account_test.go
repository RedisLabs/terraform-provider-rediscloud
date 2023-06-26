package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudCloudAccount_basic(t *testing.T) {
	name := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: providerFactories,
		CheckDestroy:             nil, // test doesn't create a resource at the moment, so don't need to check anything
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudCloudAccountDataSource, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.rediscloud_cloud_account.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_cloud_account.test", "provider_type", "AWS"),
					resource.TestCheckResourceAttr("data.rediscloud_cloud_account.test", "name", name),
					resource.TestCheckResourceAttrSet("data.rediscloud_cloud_account.test", "access_key_id"),
				),
			},
		},
	})
}

const testAccDatasourceRedisCloudCloudAccountDataSource = `
data "rediscloud_cloud_account" "test" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "%s"
}
`
