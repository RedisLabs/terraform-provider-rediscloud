package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testActiveActivePrivateLinkConfigFile = "./privatelink/testdata/testActiveActivePrivateLink.tf"

func TestAccResourceRedisCloudActiveActivePrivateLink_CRUDI(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	password := acctest.RandString(20)

	principal1 := "123456789012"
	principal2 := "234567890123"

	const resourceName = "rediscloud_private_link.private_link"
	const datasourceName = "data.rediscloud_private_link.private_link"
	shareName := acctest.RandomWithPrefix(testResourcePrefix) + "privatelink-active-active_test-share"
	terraformConfig := getRedisActiveActivePrivateLinkConfig(t, testActiveActivePrivateLinkConfigFile, shareName, password, principal1, principal2)

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
					resource.TestCheckResourceAttr(resourceName, "share_name", shareName),
					resource.TestCheckResourceAttrSet(resourceName, "principal"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(resourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(resourceName, "connections"),
					resource.TestCheckResourceAttrSet(resourceName, "databases"),

					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttr(datasourceName, "share_name", shareName),
					resource.TestCheckResourceAttrSet(datasourceName, "principal"),
					resource.TestCheckResourceAttrSet(datasourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(datasourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(datasourceName, "connections"),
					resource.TestCheckResourceAttrSet(datasourceName, "databases"),
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

func getRedisActiveActivePrivateLinkConfig(t *testing.T, testFile, shareName, password, principal1, principal2 string) string {
	subName := acctest.RandomWithPrefix(testResourcePrefix) + "-aa-private-link"
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	return fmt.Sprintf(string(content), subName, exampleCloudAccountName, shareName, password, principal1, principal2)
}
