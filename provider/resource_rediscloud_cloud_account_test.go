package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	client2 "github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"os"
	"regexp"
	"strconv"
	"testing"
)

func TestAccResourceRedisCloudCloudAccount_basic(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	if testing.Short() {
		t.Skip("Required environment variables currently not available under CI")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)

	tf := fmt.Sprintf(testAccResourceRedisCloudCloudAccount,
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_ACCESS_SECRET_KEY"),
		os.Getenv("AWS_CONSOLE_USERNAME"),
		os.Getenv("AWS_CONSOLE_PASSWORD"),
		name,
		os.Getenv("AWS_SIGNIN_URL"),
	)
	const resourceName = "rediscloud_cloud_account.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckCloudAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: tf,
				Check: resource.ComposeAggregateTestCheckFunc(
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

func testAccCheckCloudAccountDestroy(s *terraform.State) error {
	client := testProvider.Meta().(*client2.ApiClient)

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_cloud_account" {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		accounts, err := client.Client.CloudAccount.List(context.TODO())
		if err != nil {
			return err
		}

		for _, account := range accounts {
			if redis.IntValue(account.ID) == subId {
				return fmt.Errorf("account %d still exists", subId)
			}
		}
	}

	return nil
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
