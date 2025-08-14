package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudProDatabase_Upgrade(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	const resourceName = "rediscloud_subscription_database.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database and replica database creation
			{
				Config: getRedisCloudUpgradeConfig("7.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "7.2"),
				),
			},
			// Test database is updated successfully
			{
				Config: getRedisCloudUpgradeConfig("7.4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "7.4"),
				),
			},
		},
	})
}

func getRedisCloudUpgradeConfig(redisVersion string) (string, error) {
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	name := acctest.RandomWithPrefix(testResourcePrefix)

	content, err := os.ReadFile("./testdata/testAccResourceRedisCloudProDatabaseUpgrade.tf")
	if err != nil {
		return "", fmt.Errorf("could not read test data: %w", err)
	}

	return fmt.Sprintf(string(content), testCloudAccountName, name, redisVersion), nil
}
