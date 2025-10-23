package provider

import (
	"fmt"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testActiveActivePrivateLinkConfigFile = "../privatelink/testdata/active_active_private_link.tf"

func TestAccResourceRedisCloudActiveActivePrivateLink_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	password := acctest.RandString(20)

	const resourceName = "rediscloud_private_link.private_link"
	const datasourceName = "data.rediscloud_private_link.private_link"
	shareName := acctest.RandomWithPrefix(testResourcePrefix) + "privatelink-active-active_test-share"
	terraformConfig := getRedisActiveActivePrivateLinkConfig(t, testActiveActivePrivateLinkConfigFile, shareName, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: terraformConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "share_name"),
					resource.TestCheckResourceAttrSet(resourceName, "region_id"),
					resource.TestCheckResourceAttr(resourceName, "principal.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(resourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(resourceName, "connections.#"),
					resource.TestCheckResourceAttrSet(resourceName, "databases.#"),

					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "region_id"),
					resource.TestCheckResourceAttr(datasourceName, "principals.#", "2"),

					resource.TestCheckResourceAttrSet(datasourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(datasourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(datasourceName, "connections.#"),
					resource.TestCheckResourceAttrSet(datasourceName, "databases.#"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func getRedisActiveActivePrivateLinkConfig(t *testing.T, testFile, shareName, password string) string {
	subName := acctest.RandomWithPrefix(testResourcePrefix) + "-aa-private-link"
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	content := utils.GetTestConfig(t, testFile)
	return fmt.Sprintf(content, subName, exampleCloudAccountName, shareName, password)
}
