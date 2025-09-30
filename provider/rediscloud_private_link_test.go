package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testPrivateLinkConfigFile = "./privatelink/testdata/testPrivateLink.tf"

func TestAccResourceRedisCloudPrivateLink_CRUDI(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")
	testAccRequiresEnvVar(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")

	const resourceName = "rediscloud_private_link.pro_private_link"
	const datasourceName = "data.rediscloud_private_link.pro_private_link"
	const datasourceScriptName = "data.rediscloud_private_link_endpoint_script.endpoint_script"

	shareName := acctest.RandomWithPrefix(testResourcePrefix) + "-privatelink"

	terraformConfig := getRedisPrivateLinkConfig(t, shareName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: terraformConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "share_name"),
					resource.TestCheckResourceAttr(resourceName, "principal.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(resourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(resourceName, "connections"),
					resource.TestCheckResourceAttrSet(resourceName, "databases"),

					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttr(datasourceName, "principals.#", "2"),

					resource.TestCheckResourceAttrSet(datasourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(datasourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(datasourceName, "connections"),
					resource.TestCheckResourceAttrSet(datasourceName, "databases"),

					//resource.TestCheckResourceAttrSet(datasourceScriptName, "id"),
					//resource.TestCheckResourceAttrSet(datasourceScriptName, "subscription_id"),
					//resource.TestCheckResourceAttrSet(datasourceScriptName, "endpoint_script"),
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

func getRedisPrivateLinkConfig(t *testing.T, shareName string) string {
	subName := acctest.RandomWithPrefix(testResourcePrefix) + "-pro-private-link"
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	principal1 := "123456789012"
	principal2 := "234567890123"

	content := getTestConfig(t, testPrivateLinkConfigFile)
	return fmt.Sprintf(content, subName, exampleCloudAccountName, shareName, principal1, principal2)
}
