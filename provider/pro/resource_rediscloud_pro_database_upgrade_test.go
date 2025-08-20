package pro

import (
	"fmt"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudProDatabase_Upgrade(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	const resourceName = "rediscloud_subscription_database.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { utils.TestAccPreCheck(t); utils.TestAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: provider.ProviderFactories(t),
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database and replica database creation
			{
				Config: getRedisCloudUpgradeConfig(t, "7.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "7.2"),
				),
			},
			// Test database is updated successfully
			{
				Config: getRedisCloudUpgradeConfig(t, "7.4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "7.4"),
				),
			},
		},
	})
}

func getRedisCloudUpgradeConfig(t *testing.T, redisVersion string) string {
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	name := acctest.RandomWithPrefix(utils.TestResourcePrefix)

	content, err := os.ReadFile("./testdata/testAccResourceRedisCloudProDatabaseUpgrade.tf")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	return fmt.Sprintf(string(content), testCloudAccountName, name, redisVersion)
}
