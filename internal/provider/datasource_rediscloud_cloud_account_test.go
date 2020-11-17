package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudCloudAccount_basic(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)

	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	tf := fmt.Sprintf(testAccDatasourceRedisCloudCloudAccountOne,
		awsAccessKey,
		os.Getenv("AWS_ACCESS_SECRET_KEY"),
		os.Getenv("AWS_CONSOLE_USERNAME"),
		os.Getenv("AWS_CONSOLE_PASSWORD"),
		name,
		os.Getenv("AWS_SIGNIN_URL"),
	)
	resourceName := "rediscloud_cloud_account.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckCloudAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: tf,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "name", regexp.MustCompile(name)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudCloudAccountDataSource, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.rediscloud_cloud_account.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_cloud_account.test", "provider_type", "AWS"),
					resource.TestCheckResourceAttr("data.rediscloud_cloud_account.test", "name", name),
					resource.TestCheckResourceAttr("data.rediscloud_cloud_account.test", "access_key_id", awsAccessKey),
				),
			},
		},
	})
}

const testAccDatasourceRedisCloudCloudAccountOne = `
resource "rediscloud_cloud_account" "test" {
  access_key_id = "%s"
  access_secret_key = "%s"
  console_username = "%s"
  console_password = "%s"
  name = "%s"
  provider_type = "AWS"
  sign_in_login_url = "%s"
}
`

const testAccDatasourceRedisCloudCloudAccountDataSource = `
data "rediscloud_cloud_account" "test" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "%s"
}
`
