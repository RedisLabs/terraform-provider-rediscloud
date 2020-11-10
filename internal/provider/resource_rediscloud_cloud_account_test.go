package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"regexp"
	"testing"
)

func TestAccResourceRedisCloudCloudAccount(t *testing.T) {
	t.Skip("Required environment variables currently not available under CI")

	name := acctest.RandomWithPrefix("tf-test")

	tf := fmt.Sprintf(testAccResourceRedisCloudCloudAccount,
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_ACCESS_SECRET_KEY"),
		os.Getenv("AWS_CONSOLE_USERNAME"),
		os.Getenv("AWS_CONSOLE_PASSWORD"),
		name,
		os.Getenv("AWS_SIGNIN_URL"),
	)
	resourceName := "rediscloud_cloud_account.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: tf,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						resourceName, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(
						resourceName, "status", "active"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"access_secret_key", "console_username", "console_password", "sign_in_login_url"},
			},
		},
	})
}

const testAccResourceRedisCloudCloudAccount = `
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
