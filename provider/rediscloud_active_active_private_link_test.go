package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"

	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const testActiveActivePrivateLinkConfigFile = "./privatelink/testdata/active_active_private_link.tf"
const testActiveActivePrivateLinkConfigWithoutPrivateLinkFile = "./privatelink/testdata/active_active_private_link_without_privatelink.tf"

func TestAccResourceRedisCloudActiveActivePrivateLink_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const resourceName = "rediscloud_active_active_private_link.aa_private_link"
	const subscriptionResourceName = "rediscloud_active_active_subscription.aa_subscription"
	const regionsDataSourceName = "data.rediscloud_active_active_subscription_regions.aa_regions_info"
	const datasourceName = "data.rediscloud_active_active_private_link.aa_private_link"

	// Generate names reused across configs
	subName := acctest.RandomWithPrefix(testResourcePrefix) + "-aa-private-link"
	shareName := acctest.RandomWithPrefix(testResourcePrefix) + "-privatelink-aa"
	password := acctest.RandString(20)

	terraformConfig := getRedisActiveActivePrivateLinkConfigWithNames(t, subName, shareName, password)
	terraformConfigWithoutPrivateLink := getRedisActiveActivePrivateLinkConfigWithoutPrivateLink(t, subName, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create everything including privatelink
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
			// Step 2: Import test
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Remove privatelink, verify deletion via API
			{
				Config: terraformConfigWithoutPrivateLink,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(subscriptionResourceName, "id"),
					testAccCheckActiveActivePrivateLinkDeleted(subscriptionResourceName, regionsDataSourceName),
				),
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

func getRedisActiveActivePrivateLinkConfigWithNames(t *testing.T, subName, shareName, password string) string {
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	content := utils.GetTestConfig(t, testActiveActivePrivateLinkConfigFile)
	return fmt.Sprintf(content, subName, exampleCloudAccountName, shareName, password)
}

func getRedisActiveActivePrivateLinkConfigWithoutPrivateLink(t *testing.T, subName, password string) string {
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	content := utils.GetTestConfig(t, testActiveActivePrivateLinkConfigWithoutPrivateLinkFile)
	return fmt.Sprintf(content, subName, exampleCloudAccountName, password)
}

func testAccCheckActiveActivePrivateLinkDeleted(subscriptionResourceName, regionsDataSourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		subResource, ok := s.RootModule().Resources[subscriptionResourceName]
		if !ok {
			return fmt.Errorf("subscription not found: %s", subscriptionResourceName)
		}

		subId, err := strconv.Atoi(subResource.Primary.ID)
		if err != nil {
			return err
		}

		// Get region_id from the regions data source
		regionsResource, ok := s.RootModule().Resources[regionsDataSourceName]
		if !ok {
			return fmt.Errorf("regions data source not found: %s", regionsDataSourceName)
		}

		regionIdStr := regionsResource.Primary.Attributes["regions.0.region_id"]
		regionId, err := strconv.Atoi(regionIdStr)
		if err != nil {
			return fmt.Errorf("could not parse region_id: %v", err)
		}

		apiClient, err := getTestClient()
		if err != nil {
			return err
		}

		_, err = apiClient.Client.PrivateLink.GetActiveActivePrivateLink(context.TODO(), subId, regionId)
		if err == nil {
			return fmt.Errorf("active-active privatelink for subscription %d region %d still exists after deletion", subId, regionId)
		}

		var notFound *pl.NotFoundActiveActive
		if !errors.As(err, &notFound) {
			return fmt.Errorf("unexpected error checking active-active privatelink: %v", err)
		}

		return nil
	}
}
