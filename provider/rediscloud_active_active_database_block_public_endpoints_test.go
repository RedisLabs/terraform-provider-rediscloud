package provider

import (
	"fmt"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccActiveActiveSubscriptionDatabase_BlockPublicEndpoints(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_active_active_subscription_database.example"
	const datasourceName = "data.rediscloud_active_active_subscription_database.example"
	password := acctest.RandString(20)
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)

	contentDisabled := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_disabled.tf")
	configDisabled := fmt.Sprintf(contentDisabled, subscriptionName, password)

	contentEnabled := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_enabled.tf")
	configEnabled := fmt.Sprintf(contentEnabled, subscriptionName, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: configDisabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the subscription has public_endpoint_access disabled
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription.example", "public_endpoint_access", "false"),

					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),

					// Global source IPs should be explicitly set
					resource.TestCheckResourceAttr(databaseResource, "global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "global_source_ips.*", "192.168.0.0/16"),
					resource.TestCheckResourceAttr(databaseResource, "global_enable_default_user", "true"),

					// Override region 0 (us-east-1) has no override, inherits global
					resource.TestCheckResourceAttr(databaseResource, "override_region.0.override_global_source_ips.#", "0"),
					resource.TestCheckResourceAttr(databaseResource, "override_region.0.enable_default_user", "true"),

					// Override region 1 (us-east-2) has explicit override of source_ips
					resource.TestCheckResourceAttr(databaseResource, "override_region.1.override_global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "override_region.1.override_global_source_ips.*", "172.16.0.0/16"),
					resource.TestCheckResourceAttr(databaseResource, "override_region.1.enable_default_user", "true"),

					// Data source checks
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(datasourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(datasourceName, "global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "global_source_ips.*", "192.168.0.0/16"),
				),
			},
			{
				Config: configEnabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the subscription has public_endpoint_access enabled
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription.example", "public_endpoint_access", "true"),
					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),

					// Global source IPs should be the same (explicitly set in testdata)
					resource.TestCheckResourceAttr(databaseResource, "global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "global_source_ips.*", "192.168.0.0/16"),
					resource.TestCheckResourceAttr(databaseResource, "global_enable_default_user", "true"),
					// Override regions should have the same overrides
					resource.TestCheckResourceAttr(databaseResource, "override_region.1.override_global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "override_region.1.override_global_source_ips.*", "172.16.0.0/16"),
					resource.TestCheckResourceAttr(databaseResource, "override_region.1.enable_default_user", "true"),
					// Data source checks after update
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(datasourceName, "global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "global_source_ips.*", "192.168.0.0/16"),
				),
			},
		},
	})

}
