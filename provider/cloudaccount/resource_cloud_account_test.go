package cloudaccount_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccResourceRedisCloudCloudAccount_basic(t *testing.T) {
	accessKeyID, accessKeyIDCheck := envchecks.ValueAndCheck("AWS_ACCESS_KEY_ID")
	accessSecretKey, accessSecretKeyCheck := envchecks.ValueAndCheck("AWS_SECRET_ACCESS_KEY")
	consoleUsername, consoleUsernameCheck := envchecks.ValueAndCheck("AWS_CONSOLE_USERNAME")
	consolePassword, consolePasswordCheck := envchecks.ValueAndCheck("AWS_CONSOLE_PASSWORD")
	signInLoginUrl, signInLoginUrlCheck := envchecks.ValueAndCheck("AWS_SIGNIN_URL")

	name := utils.RandomWithPrefix()
	updatedName := name + "-updated"
	const resourceName = "rediscloud_cloud_account.test"

	cloudAccountConfigVariables := func(name string) config.Variables {
		return config.Variables{
			"access_key_id":     config.StringVariable(accessKeyID),
			"access_secret_key": config.StringVariable(accessSecretKey),
			"console_username":  config.StringVariable(consoleUsername),
			"console_password":  config.StringVariable(consolePassword),
			"name":              config.StringVariable(name),
			"provider_type":     config.StringVariable("AWS"),
			"sign_in_login_url": config.StringVariable(signInLoginUrl),
		}
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: envchecks.ComposePreChecks(t,
			envchecks.RedisCloudCheck,
			accessKeyIDCheck,
			accessSecretKeyCheck,
			consoleUsernameCheck,
			consolePasswordCheck,
			signInLoginUrlCheck,
		),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckCloudAccountDestroy,
		Steps: []resource.TestStep{
			{ // Create
				ConfigFile:      config.StaticFile("./testdata/resource_basic.tf"),
				ConfigVariables: cloudAccountConfigVariables(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
			{ // Update in-place (name change)
				ConfigFile:      config.StaticFile("./testdata/resource_basic.tf"),
				ConfigVariables: cloudAccountConfigVariables(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "status", "active"),
				),
			},
			{ // Import
				ConfigFile:              config.StaticFile("./testdata/resource_basic.tf"),
				ConfigVariables:         cloudAccountConfigVariables(updatedName),
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"access_secret_key", "console_username", "console_password", "sign_in_login_url"},
			},
		},
	})
}

// TestAccResourceRedisCloudCloudAccount_invalidProviderType verifies the
// provider_type OneOf validator rejects unsupported values at plan time.
func TestAccResourceRedisCloudCloudAccount_invalidProviderType(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: envchecks.ComposePreChecks(t,
			envchecks.RedisCloudCheck,
		),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"access_key_id":     config.StringVariable("dummy_access_key_id"),
					"access_secret_key": config.StringVariable("dummy_access_secret_key"),
					"console_username":  config.StringVariable("dummy_console_username"),
					"console_password":  config.StringVariable("dummy_console_password"),
					"name":              config.StringVariable("dummy_name"),
					"provider_type":     config.StringVariable("AZURE"),
					"sign_in_login_url": config.StringVariable("dummy_sign_in_login_url"),
				},
				ExpectError: regexp.MustCompile("value must be one of"),
			},
		},
	})
}

func testAccCheckCloudAccountDestroy(s *terraform.State) error {
	apiClient, err := client.GetTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_cloud_account" {
			continue
		}

		id, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		accounts, err := apiClient.Client.CloudAccount.List(context.TODO())
		if err != nil {
			return err
		}

		for _, account := range accounts {
			if redis.IntValue(account.ID) == id {
				return fmt.Errorf("account %d still exists", id)
			}
		}
	}

	return nil
}
